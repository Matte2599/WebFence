//go:build windows

package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func checkOuterBoundary(call cdpCall, session string, confined bool, job windows.Handle, sid *windows.SID, profile, outsidePath string) error {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); fmt.Fprint(w, "local fixture") }))
	defer server.Close()
	var result struct {
		Result struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"result"`
		Exception any `json:"exceptionDetails"`
	}
	expression := fmt.Sprintf(`fetch(%q,{mode:'no-cors',signal:AbortSignal.timeout(1500)}).then(()=>"reached").catch(e=>e.name)`, server.URL)
	if err := call("Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true}, session, &result); err != nil {
		return err
	}
	if result.Exception != nil || result.Result.Type != "string" {
		return errors.New("HTTP evaluation failed")
	}
	if confined {
		if hits.Load() != 0 || (result.Result.Value != "TypeError" && result.Result.Value != "TimeoutError") {
			return errors.New("confined browser reached HTTP listener")
		}
		if err := checkBrowserTokens(call, job, sid); err != nil {
			return err
		}
	} else if result.Result.Value != "reached" || hits.Load() == 0 {
		return errors.New("browser HTTP positive control failed")
	}
	// Use the exact file already read by the unrestricted browser control.
	content, err := os.ReadFile(outsidePath)
	if err != nil || string(content) != "<p>outside fixture</p>" {
		return errors.New("outside fixture unavailable to parent")
	}
	insidePath := filepath.Join(profile, "fixture.html")
	if err := os.WriteFile(insidePath, []byte("<p>inside fixture</p>"), 0600); err != nil {
		return err
	}
	navigate := func(path string) (string, error) {
		address := url.URL{Scheme: "file", Path: "/" + filepath.ToSlash(path)}
		var navigation struct {
			Error string `json:"errorText"`
		}
		err := call("Page.navigate", map[string]any{"url": address.String()}, session, &navigation)
		return navigation.Error, err
	}
	denied, err := navigate(outsidePath)
	if err != nil {
		return err
	}
	if (confined && denied != "net::ERR_ACCESS_DENIED" && denied != "net::ERR_FILE_NOT_FOUND") || (!confined && denied != "") {
		return fmt.Errorf("outside file boundary mismatch: %s", denied)
	}
	if denied, err = navigate(insidePath); err != nil || denied != "" {
		return fmt.Errorf("private file control failed: %s %v", denied, err)
	}
	for i := 0; i < 20; i++ {
		result = struct {
			Result struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"result"`
			Exception any `json:"exceptionDetails"`
		}{}
		if err := call("Runtime.evaluate", map[string]any{"expression": "document.readyState==='complete'?document.body.textContent:''", "returnByValue": true}, session, &result); err != nil {
			return err
		}
		if result.Exception == nil && result.Result.Type == "string" && result.Result.Value == "inside fixture" {
			break
		}
		if i == 19 {
			return errors.New("private fixture DOM failed")
		}
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Printf("PASS browser boundary control: confined=%v direct HTTP/file policy, private file readable\n", confined)
	return nil
}

func checkBrowserTokens(call cdpCall, job windows.Handle, sid *windows.SID) error {
	var result struct {
		Processes []struct {
			Type string `json:"type"`
			ID   uint32 `json:"id"`
		} `json:"processInfo"`
	}
	if err := call("SystemInfo.getProcessInfo", map[string]any{}, "", &result); err != nil {
		return err
	}
	kinds := map[string]bool{}
	for _, entry := range result.Processes {
		kinds[entry.Type] = true
		process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, entry.ID)
		if err != nil {
			return err
		}
		err = checkContainerProcess(process, job, sid)
		windows.CloseHandle(process)
		if err != nil {
			return fmt.Errorf("browser descendant %s: %w", entry.Type, err)
		}
	}
	for _, kind := range []string{"browser", "renderer", "GPU", "network.mojom.NetworkService"} {
		if !kinds[kind] {
			return fmt.Errorf("missing descendant %s", kind)
		}
	}
	return nil
}

func checkContainerProcess(process, job windows.Handle, sid *windows.SID) error {
	var inJob int32
	result, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("IsProcessInJob").Call(uintptr(process), uintptr(job), uintptr(unsafe.Pointer(&inJob)))
	if result == 0 || inJob == 0 {
		return fmt.Errorf("descendant outside job: %v", err)
	}
	var token windows.Token
	if err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token); err != nil {
		return err
	}
	defer token.Close()
	var actual, size uint32
	if err := windows.GetTokenInformation(token, tokenIsAppContainer, (*byte)(unsafe.Pointer(&actual)), 4, &size); err != nil || actual != 1 {
		return fmt.Errorf("descendant outside AppContainer: %v", err)
	}
	// TOKEN_GROUPS for TokenCapabilities: the first DWORD is GroupCount.
	var data [128]uintptr
	if err := windows.GetTokenInformation(token, 30, (*byte)(unsafe.Pointer(&data[0])), uint32(unsafe.Sizeof(data)), &size); err != nil {
		return err
	}
	if *(*uint32)(unsafe.Pointer(&data[0])) != 0 {
		return errors.New("descendant has capabilities")
	}
	// TOKEN_APPCONTAINER_INFORMATION begins with a SID pointer into the buffer.
	if err := windows.GetTokenInformation(token, 31, (*byte)(unsafe.Pointer(&data[0])), uint32(unsafe.Sizeof(data)), &size); err != nil {
		return err
	}
	actualSID := *(**windows.SID)(unsafe.Pointer(&data[0]))
	if actualSID == nil || !actualSID.Equals(sid) {
		return errors.New("descendant has different AppContainer SID")
	}
	return nil
}
