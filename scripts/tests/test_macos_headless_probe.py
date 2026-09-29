import importlib.util
import json
import os
from pathlib import Path
import sys
import unittest


@unittest.skipUnless(sys.platform != 'win32', 'POSIX pipes')
class HeadlessProtocolTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        path = Path(__file__).resolve().parents[2] / 'experiments/m3-macos-sandbox/headless_probe.py'
        spec = importlib.util.spec_from_file_location('headless_probe', path)
        cls.probe = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(cls.probe)

    def response(self, frames, session=None, limit=None):
        read_fd, writer = os.pipe()
        output = os.open(os.devnull, os.O_WRONLY)
        # Small frames avoid blocking the writer before the reader runs.
        os.write(writer, frames)
        os.close(writer)
        cdp = self.probe.CDP(read_fd, output)
        if limit:
            cdp.deadline = 0
        try:
            return cdp.call('Runtime.evaluate', session=session)
        finally:
            os.close(read_fd)
            os.close(output)

    def frame(self, reply):
        return json.dumps(reply).encode() + b'\0'

    def test_event_then_valid_session(self):
        frames = self.frame({'method': 'Page.loadEventFired'}) + self.frame({
            'id': 1, 'sessionId': 'fixture', 'result': {'ok': True}})
        self.assertEqual(self.response(frames, 'fixture'), {'ok': True})

    def test_closed_pipe(self):
        with self.assertRaisesRegex(RuntimeError, 'EOF'):
            self.response(b'')

    def test_protocol_error(self):
        with self.assertRaisesRegex(RuntimeError, 'rejected'):
            self.response(self.frame({'id': 1, 'error': {'message': 'fixture'}}))

    def test_missing_result(self):
        with self.assertRaisesRegex(RuntimeError, 'rejected'):
            self.response(self.frame({'id': 1}))

    def test_wrong_session(self):
        with self.assertRaisesRegex(RuntimeError, 'session mismatch'):
            self.response(self.frame({'id': 1, 'result': {}, 'sessionId': 'other'}), 'fixture')

    def test_deadline(self):
        with self.assertRaisesRegex(RuntimeError, 'deadline'):
            self.response(b'partial', limit=True)

    def test_frame_limit(self):
        from unittest.mock import patch
        with patch.object(self.probe, 'MAX_FRAME', 1024):
            with self.assertRaisesRegex(RuntimeError, 'frame limit'):
                self.response(b'x' * 1025)

    def test_event_limit(self):
        with self.assertRaisesRegex(RuntimeError, 'event limit'):
            self.response(b'{}\0' * 128)

    def test_javascript_exception(self):
        with self.assertRaisesRegex(RuntimeError, 'JavaScript exception'):
            self.probe.value({'exceptionDetails': {'text': 'fixture'}})

    def test_wrong_javascript_type(self):
        with self.assertRaisesRegex(RuntimeError, 'result type'):
            self.probe.value({'result': {'type': 'boolean', 'value': True}})
