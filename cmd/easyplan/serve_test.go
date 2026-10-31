package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// The names of the environment variables that tell the test binary that it
// runs as the helper of the test of the server.
const (
	helperAddress = "EASYPLAN_TEST_HELPER_ADDRESS"
	helperData    = "EASYPLAN_TEST_HELPER_DATA"
)

// TestMain runs the tests of the package.
//
// The helper of the test of the server is the same binary, so the helper runs
// before the tests.
func TestMain(m *testing.M) {
	if addr := os.Getenv(helperAddress); addr != "" {
		os.Exit(helperServe(addr))
	}
	os.Exit(m.Run())
}

// helperServe runs the server of the configuration of the environment and
// reports the result as the exit code of the process.
//
// The server runs in a process of its own: it writes the time zone of the
// process while it starts. A test that runs the server in the same process
// would read the time zone of the process at the same time, and the race
// detector would report it.
//
// The end of the input of the process stops the server, the parent closes it
// when the server shall stop. The exit code 0 means that the server stopped
// without an error.
func helperServe(addr string) int {
	data := os.Getenv(helperData)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		if _, err := reader.ReadString('\n'); err != nil {
			cancel()
		}
	}()

	if err := serve(ctx, config{addr: addr, data: data, tz: defaultTZ}); err != nil {
		fmt.Fprintln(os.Stderr, "serve:", err)
		return 1
	}
	return 0
}

// freeAddress returns an address of the loopback interface that no server
// uses at the moment.
//
// The listener is closed right away, so the port is free. Another process
// could take it in the meantime, the chance is small.
func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() failed: %v", err)
	}
	return addr
}

// startServer starts the server in a process of its own.
//
// The function returns the command of the process, the way to stop the server
// and the output of the process with the details of a failure.
func startServer(t *testing.T, addr, data string) (*exec.Cmd, io.WriteCloser, *bytes.Buffer) {
	t.Helper()

	output := &bytes.Buffer{}
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), helperAddress+"="+addr, helperData+"="+data)
	cmd.Stderr = output

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("cmd.StdinPipe() failed: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start() failed: %v", err)
	}
	t.Cleanup(func() {
		stdin.Close()
	})

	return cmd, stdin, output
}

// waitForServer waits until the server answers a request.
//
// The wait ends when the timeout is over, so the test cannot hang. The server
// has to answer on the loopback interface, so the test does not need the
// network.
func waitForServer(t *testing.T, addr string) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	timeout := time.NewTimer(15 * time.Second)
	defer timeout.Stop()

	for {
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("the status of the answer is %d, want %d", resp.StatusCode, http.StatusOK)
			}
			return
		}

		select {
		case <-timeout.C:
			t.Fatalf("the server does not answer on %s: %v", addr, err)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// TestServe_starts_and_stops_the_server checks that the server starts, that
// it answers on the address of the configuration and that it stops without an
// error when the context is done.
func TestServe_starts_and_stops_the_server(t *testing.T) {
	addr := freeAddress(t)
	data := t.TempDir()

	cmd, stdin, output := startServer(t, addr, data)

	waitForServer(t, addr)

	// The end of the input of the process stops the server.
	if err := stdin.Close(); err != nil {
		t.Errorf("cannot close the input of the server: %v", err)
	}

	// The exit code 0 means that the server stopped without an error.
	if err := cmd.Wait(); err != nil {
		t.Errorf("the server failed with %v, want no error:\n%s", err, output.String())
	}
}

// TestServe_reports_an_occupied_port checks that the server does not start
// on a port that another server uses.
func TestServe_reports_an_occupied_port(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() failed: %v", err)
	}
	defer listener.Close()

	cmd, _, output := startServer(t, listener.Addr().String(), t.TempDir())

	err = cmd.Wait()
	if err == nil {
		t.Fatal("the server started on the occupied port, want an error")
	}
	if !strings.Contains(output.String(), "address already in use") {
		t.Errorf("the server failed with %q, want an error about the occupied port", output.String())
	}
}

// TestServe_writes_the_database_in_the_data_directory checks that the server
// keeps its data in the directory of the configuration.
func TestServe_writes_the_database_in_the_data_directory(t *testing.T) {
	data := t.TempDir()

	cmd, stdin, output := startServer(t, "127.0.0.1:0", data)

	if err := stdin.Close(); err != nil {
		t.Errorf("cannot close the input of the server: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Errorf("the server failed with %v, want no error:\n%s", err, output.String())
	}

	entries, err := readDirNames(data)
	if err != nil {
		t.Fatalf("cannot read the data directory: %v", err)
	}
	found := false
	for _, name := range entries {
		if name == databaseName {
			found = true
		}
	}
	if !found {
		t.Errorf("the data directory holds %v, want the file %q", entries, databaseName)
	}
}

// readDirNames returns the names of the files of a directory.
func readDirNames(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Name())
	}
	return names, nil
}
