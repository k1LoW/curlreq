package curlreq_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/curlreq"
)

// TestParse ref: https://github.com/tj/parse-curl.js
func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  *curlreq.Parsed
	}{
		{
			`curl http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{},
			},
		},
		{
			`curl -I http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodHead,
				Header: http.Header{},
			},
		},
		{
			`curl -I http://api.sloths.com -vvv --foo --whatever bar`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodHead,
				Header: http.Header{},
			},
		},
		{
			`curl -H "Origin: https://example.com" https://example.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://example.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Origin": []string{"https://example.com"},
				},
			},
		},
		{
			`curl --compressed http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept-Encoding": []string{"deflate, gzip"},
				},
			},
		},
		{
			`curl -H "Accept-Encoding: gzip" --compressed http://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept-Encoding": []string{"gzip"},
				},
			},
		},
		{
			`curl -X DELETE http://api.sloths.com/sloth/4`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com/sloth/4"),
				Method: http.MethodDelete,
				Header: http.Header{},
			},
		},
		{
			`curl -XPUT http://api.sloths.com/sloth/4`,
			&curlreq.Parsed{
				URL:    URL(t, "http://api.sloths.com/sloth/4"),
				Method: http.MethodPut,
				Header: http.Header{},
			},
		},
		{
			`curl https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{},
			},
		},
		{
			`curl -u tobi:ferret https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Authorization": []string{"Basic dG9iaTpmZXJyZXQ="},
				},
			},
		},
		{
			`curl -d "foo=bar" https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("foo=bar"),
			},
		},
		{
			`curl -d "foo=bar" -d bar=baz https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("foo=bar&bar=baz"),
			},
		},
		{
			`curl -H "Accept: text/plain" --header "User-Agent: slothy" https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/plain"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -H 'Accept: text/*' --header 'User-Agent: slothy' https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/*"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -H 'Accept: text/*' -A slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/*"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
		{
			`curl -b 'foo=bar' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"foo=bar"},
				},
			},
		},
		{
			`curl --cookie 'foo=bar' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"foo=bar"},
				},
			},
		},
		{
			`curl --cookie 'species=sloth;type=galactic' slothy https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Cookie": []string{"species=sloth;type=galactic"},
				},
			},
		},
		{
			`curl -H 'Accept:text/*' --header 'User-Agent:slothy' https://api.sloths.com`,
			&curlreq.Parsed{
				URL:    URL(t, "https://api.sloths.com"),
				Method: http.MethodGet,
				Header: http.Header{
					"Accept":     []string{"text/*"},
					"User-Agent": []string{"slothy"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := curlreq.Parse(tt.input)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Errorf("%s", diff)
			}
		})
	}
}

func TestParseExtra(t *testing.T) {
	tests := []struct {
		input string
		want  *curlreq.Parsed
	}{
		{
			`curl example.com`,
			&curlreq.Parsed{
				URL:    nil,
				Method: http.MethodGet,
				Header: http.Header{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := curlreq.Parse(tt.input)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(got, tt.want, nil); diff != "" {
				t.Errorf("%s", diff)
			}
			_, _ = got.Request()
		})
	}
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			`curl http://example.com`,
			`{"url":"http://example.com","method":"GET","header":{}}`,
		},
		{
			`curl 'http://google.com/' \
  -H 'Accept-Encoding: gzip, deflate, sdch' \
  -H 'Accept-Language: en-US,en;q=0.8,da;q=0.6' \
p  -H 'Upgrade-Insecure-Requests: 1' \
  -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.110 Safari/537.36' \
  -H 'Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8' \
  -H 'Connection: keep-alive' \
  --compressed`,
			`{"url":"http://google.com/","method":"GET","header":{"Accept":["text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"],"Accept-Encoding":["gzip, deflate, sdch"],"Accept-Language":["en-US,en;q=0.8,da;q=0.6"],"Connection":["keep-alive"],"Upgrade-Insecure-Requests":["1"],"User-Agent":["Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.110 Safari/537.36"]}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, err := curlreq.Parse(tt.input)
			if err != nil {
				t.Error(err)
			}
			got, err := json.Marshal(p)
			if err != nil {
				t.Error(err)
			}
			if diff := cmp.Diff(string(got), tt.want, nil); diff != "" {
				t.Errorf("%s", diff)
			}
		})
	}

	t.Run("marshal JSON with valid UTF-8 body", func(t *testing.T) {
		t.Parallel()

		p := &curlreq.Parsed{
			URL:    URL(t, "https://api.example.com"),
			Method: http.MethodPost,
			Header: http.Header{},
			Body:   []byte(`{"message":"hello"}`),
		}

		got, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("json.Marshal returned error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(got, &result); err != nil {
			t.Fatalf("json.Unmarshal returned error: %v", err)
		}

		// Valid UTF-8 should use plain encoding
		if enc, ok := result["body_encoding"]; ok && enc != "plain" {
			t.Errorf("expected body_encoding to be 'plain', got %v", enc)
		}

		if body, ok := result["body"]; !ok || body != `{"message":"hello"}` {
			t.Errorf("expected body to be preserved, got %v", body)
		}
	})

	t.Run("marshal JSON with binary body", func(t *testing.T) {
		t.Parallel()

		// Create binary data with invalid UTF-8
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		p := &curlreq.Parsed{
			URL:    URL(t, "https://api.example.com"),
			Method: http.MethodPost,
			Header: http.Header{},
			Body:   binaryData,
		}

		got, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("json.Marshal returned error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(got, &result); err != nil {
			t.Fatalf("json.Unmarshal returned error: %v", err)
		}

		// Binary data should use base64 encoding
		if enc, ok := result["body_encoding"]; !ok || enc != "base64" {
			t.Errorf("expected body_encoding to be 'base64', got %v", enc)
		}

		// Decode base64 and verify it matches original binary data
		if bodyStr, ok := result["body"].(string); ok {
			decoded, err := base64.StdEncoding.DecodeString(bodyStr)
			if err != nil {
				t.Fatalf("failed to decode base64 body: %v", err)
			}
			if diff := cmp.Diff(binaryData, decoded); diff != "" {
				t.Errorf("binary data not preserved (-want +got):\n%s", diff)
			}
		} else {
			t.Error("body field is not a string")
		}
	})
}

func Example() {
	cmd := "curl https://example.com"
	req, err := curlreq.NewRequest(cmd)
	if err != nil {
		log.Fatal(err)
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	fmt.Println(resp.StatusCode)

	// Output:
	// 200
}

func URL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestParseWithDataFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		build   func(path string) string
		want    *curlreq.Parsed
	}{
		{
			name:    "parse with -d @file",
			content: []byte(`{"key":"value"}`),
			build: func(path string) string {
				return fmt.Sprintf(`curl -d @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`{"key":"value"}`),
			},
		},
		{
			name:    "parse with --data @file",
			content: []byte(`foo=bar&baz=qux`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`foo=bar&baz=qux`),
			},
		},
		{
			name:    "parse with --data-binary @file",
			content: []byte(`binary content here`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-binary @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`binary content here`),
			},
		},
		{
			name:    "parse with inline -d@file",
			content: []byte(`{"message":"hello"}`),
			build: func(path string) string {
				return fmt.Sprintf(`curl -d@%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`{"message":"hello"}`),
			},
		},
		{
			name:    "parse with --data-ascii @file",
			content: []byte(`test data`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-ascii @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`test data`),
			},
		},
		{
			name:    "parse with inline --data=@file",
			content: []byte(`inline content`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data=@%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`inline content`),
			},
		},
		{
			name:    "parse with inline --data-binary=@file",
			content: []byte(`binary inline`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-binary=@%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`binary inline`),
			},
		},
		{
			name:    "parse with inline --data-ascii=@file",
			content: []byte(`ascii inline`),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-ascii=@%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte(`ascii inline`),
			},
		},
		{
			name: "parse with binary data containing NUL bytes",
			content: []byte{
				0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, // Binary data with NUL and high bytes
				0x00, 0x00, // More NUL bytes
				0x48, 0x65, 0x6C, 0x6C, 0x6F, // "Hello"
			},
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-binary @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0x00, 0x00, 0x48, 0x65, 0x6C, 0x6C, 0x6F},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "data.txt")
			if err := os.WriteFile(path, tt.content, 0o600); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			got, err := curlreq.Parse(tt.build(path))
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}

	t.Run("missing file returns error", func(t *testing.T) {
		t.Parallel()

		_, err := curlreq.Parse(`curl -d @does-not-exist https://api.example.com`)
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("binary data can be converted to http.Request", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "binary.dat")
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		if err := os.WriteFile(path, binaryData, 0o600); err != nil {
			t.Fatalf("failed to write binary file: %v", err)
		}

		cmd := fmt.Sprintf(`curl --data-binary @%s https://api.example.com`, path)
		parsed, err := curlreq.Parse(cmd)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		// Convert to http.Request
		req, err := parsed.Request()
		if err != nil {
			t.Fatalf("Request() returned error: %v", err)
		}

		// Read body and verify binary data is preserved
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if diff := cmp.Diff(binaryData, body); diff != "" {
			t.Errorf("binary data not preserved (-want +got):\n%s", diff)
		}
	})
}

func TestWithWorkingDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupPath func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "valid directory",
			setupPath: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "empty path returns error",
			setupPath: func(t *testing.T) string {
				return ""
			},
			wantErr: true,
		},
		{
			name: "non-existent path returns error",
			setupPath: func(t *testing.T) string {
				return "/does/not/exist/path/for/testing"
			},
			wantErr: true,
		},
		{
			name: "file path returns error",
			setupPath: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, "file.txt")
				if err := os.WriteFile(filePath, []byte("content"), 0o600); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return filePath
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.setupPath(t)
			parser, err := curlreq.NewParser(curlreq.WithWorkingDirectory(path))

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				if parser == nil {
					t.Error("expected parser, got nil")
				}
			}
		})
	}
}

func TestParserWithWorkingDirectory(t *testing.T) {
	t.Parallel()

	t.Run("relative path resolved against working directory", func(t *testing.T) {
		t.Parallel()

		// Create directory structure:
		// tempdir/
		//   subdir/
		//     data.json
		dir := t.TempDir()
		subdir := filepath.Join(dir, "subdir")
		if err := os.Mkdir(subdir, 0o755); err != nil {
			t.Fatalf("failed to create subdir: %v", err)
		}

		dataFile := filepath.Join(subdir, "data.json")
		content := []byte(`{"key":"value"}`)
		if err := os.WriteFile(dataFile, content, 0o600); err != nil {
			t.Fatalf("failed to write data file: %v", err)
		}

		// Parse with working directory set to tempdir
		parser, err := curlreq.NewParser(curlreq.WithWorkingDirectory(dir))
		if err != nil {
			t.Fatalf("failed to create parser: %v", err)
		}

		// Use relative path from working directory
		got, err := parser.Parse(`curl -d @subdir/data.json https://api.example.com`)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		want := &curlreq.Parsed{
			URL:    URL(t, "https://api.example.com"),
			Method: http.MethodPost,
			Header: http.Header{},
			Body:   content,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	t.Run("absolute path not affected by working directory", func(t *testing.T) {
		t.Parallel()

		// Create file with absolute path
		dir := t.TempDir()
		dataFile := filepath.Join(dir, "data.json")
		content := []byte(`{"absolute":"path"}`)
		if err := os.WriteFile(dataFile, content, 0o600); err != nil {
			t.Fatalf("failed to write data file: %v", err)
		}

		// Create a different working directory
		wdDir := t.TempDir()
		parser, err := curlreq.NewParser(curlreq.WithWorkingDirectory(wdDir))
		if err != nil {
			t.Fatalf("failed to create parser: %v", err)
		}

		// Use absolute path - should work regardless of working directory
		cmd := fmt.Sprintf(`curl -d @%s https://api.example.com`, dataFile)
		got, err := parser.Parse(cmd)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		want := &curlreq.Parsed{
			URL:    URL(t, "https://api.example.com"),
			Method: http.MethodPost,
			Header: http.Header{},
			Body:   content,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	t.Run("relative path with default working directory", func(t *testing.T) {
		t.Parallel()

		// Create file in current directory
		dir := t.TempDir()
		dataFile := filepath.Join(dir, "data.json")
		content := []byte(`{"default":"wd"}`)
		if err := os.WriteFile(dataFile, content, 0o600); err != nil {
			t.Fatalf("failed to write data file: %v", err)
		}

		// Parse without specifying working directory (uses ".")
		parser, err := curlreq.NewParser()
		if err != nil {
			t.Fatalf("failed to create parser: %v", err)
		}

		// Use absolute path since we can't rely on current directory in tests
		cmd := fmt.Sprintf(`curl -d @%s https://api.example.com`, dataFile)
		got, err := parser.Parse(cmd)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		if diff := cmp.Diff(content, got.Body); diff != "" {
			t.Errorf("unexpected body (-want +got):\n%s", diff)
		}
	})

	t.Run("multiple data files with working directory", func(t *testing.T) {
		t.Parallel()

		// Create directory with multiple files
		dir := t.TempDir()
		file1 := filepath.Join(dir, "file1.txt")
		file2 := filepath.Join(dir, "file2.txt")

		if err := os.WriteFile(file1, []byte("data1"), 0o600); err != nil {
			t.Fatalf("failed to write file1: %v", err)
		}
		if err := os.WriteFile(file2, []byte("data2"), 0o600); err != nil {
			t.Fatalf("failed to write file2: %v", err)
		}

		parser, err := curlreq.NewParser(curlreq.WithWorkingDirectory(dir))
		if err != nil {
			t.Fatalf("failed to create parser: %v", err)
		}

		got, err := parser.Parse(`curl -d @file1.txt -d @file2.txt https://api.example.com`)
		if err != nil {
			t.Fatalf("Parse returned error: %v", err)
		}

		want := &curlreq.Parsed{
			URL:    URL(t, "https://api.example.com"),
			Method: http.MethodPost,
			Header: http.Header{},
			Body:   []byte("data1&data2"),
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})
}

func TestParseWithDataUrlEncode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content []byte
		build   func(path string) string
		want    *curlreq.Parsed
	}{
		{
			name:    "URL encode simple content",
			content: nil, // no file needed
			build: func(path string) string {
				return `curl --data-urlencode "hello world" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("hello+world"),
			},
		},
		{
			name:    "URL encode with = prefix",
			content: nil,
			build: func(path string) string {
				return `curl --data-urlencode "=hello world" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("=hello+world"),
			},
		},
		{
			name:    "URL encode with name=content format",
			content: nil,
			build: func(path string) string {
				return `curl --data-urlencode "name=hello world" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("name=hello+world"),
			},
		},
		{
			name:    "URL encode special characters",
			content: nil,
			build: func(path string) string {
				return `curl --data-urlencode "key=value&other=data" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("key=value%26other%3Ddata"),
			},
		},
		{
			name:    "URL encode file contents with @file",
			content: []byte("hello world"),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-urlencode @%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("hello+world"),
			},
		},
		{
			name:    "URL encode file contents with name@file",
			content: []byte("hello world"),
			build: func(path string) string {
				return fmt.Sprintf(`curl --data-urlencode name@%s https://api.example.com`, path)
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("name=hello+world"),
			},
		},
		{
			name:    "URL encode inline format",
			content: nil,
			build: func(path string) string {
				return `curl --data-urlencode="test data" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("test+data"),
			},
		},
		{
			name:    "URL encode multiple parameters",
			content: nil,
			build: func(path string) string {
				return `curl --data-urlencode "name=John Doe" --data-urlencode "city=New York" https://api.example.com`
			},
			want: &curlreq.Parsed{
				URL:    URL(t, "https://api.example.com"),
				Method: http.MethodPost,
				Header: http.Header{},
				Body:   []byte("name=John+Doe&city=New+York"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var path string
			if tt.content != nil {
				dir := t.TempDir()
				path = filepath.Join(dir, "data.txt")
				if err := os.WriteFile(path, tt.content, 0o600); err != nil {
					t.Fatalf("failed to write temp file: %v", err)
				}
			}

			got, err := curlreq.Parse(tt.build(path))
			if err != nil {
				t.Fatalf("Parse returned error: %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}
