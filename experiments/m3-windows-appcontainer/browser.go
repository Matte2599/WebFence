//go:build windows

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// browserTrial keeps the browser's own sandbox enabled. The control establishes
// that this browser accepts the inherited CDP handles before testing AppContainer.
func browserTrial(executable string, sid *windows.SID) error {
	if !filepath.IsAbs(executable) {
		return errors.New("absolute browser path required")
	}
	if info, err := os.Stat(executable); err != nil || !info.Mode().IsRegular() {
		return errors.New("browser executable unavailable")
	}
	for _, confined := range []bool{false, true} {
		label := "control"
		if confined {
			label = "appcontainer"
		}
		if err := runBrowser(executable, sid, confined); err != nil {
			return fmt.Errorf("browser %s: %w", label, err)
		}
		fmt.Printf("PASS M3 Windows browser %s: CDP version and synthetic DOM script; job limited\n", label)
	}
	return nil
}

func runBrowser(executable string, sid *windows.SID, confined bool) error {
	profile, err := os.MkdirTemp("", "wf-m3-browser-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(profile)
	// Only this disposable directory receives an ACE for this unique AppContainer.
	sd, err := windows.GetNamedSecurityInfo(profile, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	oldACL, _, err := sd.DACL()
	if err != nil {
		return err
	}
	acl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.GENERIC_ALL, AccessMode: windows.GRANT_ACCESS,
		Inheritance: windows.SUB_CONTAINERS_AND_OBJECTS_INHERIT,
		Trustee:     windows.TRUSTEE{TrusteeForm: windows.TRUSTEE_IS_SID, TrusteeType: windows.TRUSTEE_IS_UNKNOWN, TrusteeValue: windows.TrusteeValueFromSID(sid)},
	}}, oldACL)
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(profile, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
		return err
	}
	// The low integrity AppContainer needs a low integrity writable scratch area.
	low, err := windows.SecurityDescriptorFromString("S:(ML;OICI;NW;;;LW)")
	if err != nil {
		return err
	}
	sacl, _, err := low.SACL()
	if err != nil {
		return err
	}
	if err := windows.SetNamedSecurityInfo(profile, windows.SE_FILE_OBJECT, windows.LABEL_SECURITY_INFORMATION, nil, nil, nil, sacl); err != nil {
		return err
	}

	browserRead, parentWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer browserRead.Close()
	defer parentWrite.Close()
	parentRead, browserWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer parentRead.Close()
	defer browserWrite.Close()
	logRead, logWrite, err := os.Pipe()
	if err != nil {
		return err
	}
	defer logRead.Close()
	defer logWrite.Close()
	handles := []windows.Handle{windows.Handle(browserRead.Fd()), windows.Handle(browserWrite.Fd()), windows.Handle(logWrite.Fd())}
	for _, handle := range handles {
		if err := windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			return err
		}
	}
	count := uint32(1)
	if confined {
		count++
	}
	attributes, err := windows.NewProcThreadAttributeList(count)
	if err != nil {
		return err
	}
	defer attributes.Delete()
	if err := attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		return err
	}
	capabilities := securityCapabilities{appContainerSID: sid}
	if confined {
		if err := attributes.Update(securityCapabilitiesAttribute, unsafe.Pointer(&capabilities), unsafe.Sizeof(capabilities)); err != nil {
			return err
		}
	}
	args := []string{executable, "--headless=new", "--no-first-run", "--no-default-browser-check", "--disable-background-networking", "--disable-component-update", "--disable-sync", "--disable-extensions", "--disable-breakpad", "--disable-crash-reporter", "--host-resolver-rules=MAP * ~NOTFOUND", "--remote-debugging-pipe", fmt.Sprintf("--remote-debugging-io-pipes=%d,%d", handles[0], handles[1]), "--user-data-dir=" + profile, "--enable-logging=stderr", "about:blank"}
	for i := range args {
		args[i] = windows.EscapeArg(args[i])
	}
	command16, err := windows.UTF16PtrFromString(strings.Join(args, " "))
	if err != nil {
		return err
	}
	executable16, _ := windows.UTF16PtrFromString(executable)
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput, startup.StdOutput, startup.StdErr = handles[0], handles[2], handles[2]
	var process windows.ProcessInformation
	if err := windows.CreateProcess(executable16, command16, nil, nil, true, windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_SUSPENDED|windows.CREATE_NO_WINDOW, nil, nil, &startup.StartupInfo, &process); err != nil {
		return fmt.Errorf("CreateProcess: %w", err)
	}
	defer windows.CloseHandle(process.Thread)
	defer windows.CloseHandle(process.Process)
	defer windows.TerminateProcess(process.Process, 1)
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 16
	limits.JobMemoryLimit = 1024 << 20
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		return err
	}
	if err := windows.AssignProcessToJobObject(job, process.Process); err != nil {
		return err
	}
	// Check the actual suspended browser token, not just the launcher canary.
	var token windows.Token
	if err := windows.OpenProcessToken(process.Process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	var isContainer, size uint32
	err = windows.GetTokenInformation(token, tokenIsAppContainer, (*byte)(unsafe.Pointer(&isContainer)), uint32(unsafe.Sizeof(isContainer)), &size)
	token.Close()
	if err != nil || (isContainer != 0) != confined {
		return fmt.Errorf("unexpected browser token: container=%d error=%v", isContainer, err)
	}
	if _, err := windows.ResumeThread(process.Thread); err != nil {
		return err
	}
	browserRead.Close()
	browserWrite.Close()
	logWrite.Close()
	logs := make(chan string, 1)
	go func() { data, _ := io.ReadAll(io.LimitReader(logRead, 8192)); logs <- string(data) }()
	done := make(chan error, 1)
	go func() { done <- checkBrowserCDP(parentRead, parentWrite) }()
	select {
	case err = <-done:
	case <-time.After(20 * time.Second):
		err = errors.New("CDP deadline exceeded")
	}
	var exitCode uint32
	windows.GetExitCodeProcess(process.Process, &exitCode)
	// Stop all descendants before inspecting logs and deleting the profile.
	windows.TerminateJobObject(job, 1)
	windows.WaitForSingleObject(process.Process, 5000)
	if err != nil {
		select {
		case log := <-logs:
			fmt.Printf("Browser diagnostic (bounded): %s\n", log)
		case <-time.After(time.Second):
		}
		return fmt.Errorf("%w (exit=0x%x)", err, exitCode)
	}
	return nil
}

func checkBrowserCDP(input io.Reader, output io.Writer) error {
	reader := bufio.NewReaderSize(input, 64<<10)
	var nextID int
	call := func(method string, params any, session string, result any) error {
		nextID++
		request := map[string]any{"id": nextID, "method": method, "params": params}
		if session != "" {
			request["sessionId"] = session
		}
		data, err := json.Marshal(request)
		if err != nil {
			return err
		}
		if _, err := output.Write(append(data, 0)); err != nil {
			return err
		}
		for range 128 {
			frame, err := reader.ReadSlice(0)
			if err != nil {
				return fmt.Errorf("%s: %w", method, err)
			}
			var reply struct {
				ID     int             `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  json.RawMessage `json:"error"`
			}
			if err := json.Unmarshal(frame[:len(frame)-1], &reply); err != nil {
				return err
			}
			if reply.ID != nextID {
				continue
			}
			if len(reply.Error) != 0 {
				return fmt.Errorf("%s: protocol error", method)
			}
			return json.Unmarshal(reply.Result, result)
		}
		return errors.New("CDP message budget exceeded")
	}
	var version struct {
		Product string `json:"product"`
	}
	if err := call("Browser.getVersion", struct{}{}, "", &version); err != nil {
		return err
	}
	if version.Product == "" {
		return errors.New("missing browser product")
	}
	fmt.Printf("Browser CDP product: %s\n", version.Product)
	var target struct {
		ID string `json:"targetId"`
	}
	if err := call("Target.createTarget", map[string]any{"url": "about:blank"}, "", &target); err != nil {
		return err
	}
	var session struct {
		ID string `json:"sessionId"`
	}
	if err := call("Target.attachToTarget", map[string]any{"targetId": target.ID, "flatten": true}, "", &session); err != nil {
		return err
	}
	var result struct {
		Result struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := call("Runtime.evaluate", map[string]any{"expression": "document.body.innerHTML='<button id=probe>local fixture</button>'; document.querySelector('#probe').textContent", "returnByValue": true}, session.ID, &result); err != nil {
		return err
	}
	if result.Result.Type != "string" || result.Result.Value != "local fixture" {
		return errors.New("synthetic DOM evaluation failed")
	}
	return nil
}
