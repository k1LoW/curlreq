package curlreq

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/mattn/go-shellwords"
)

const (
	stateBlank  = ""
	stateHeader = "header"
	stateUA     = "user-agent"
	stateData   = "data"
	stateUser   = "user"
	stateMethod = "method"
	stateCookie = "cookie"
)

type Parsed struct {
	URL    *url.URL
	Method string
	Header http.Header
	Body   string
}

// NewRequest returns *http.Request created by parsing a curl command.
func NewRequest(cmd ...string) (*http.Request, error) {
	p, err := Parse(cmd...)
	if err != nil {
		return nil, err
	}
	return p.Request()
}

// Parse a curl command.
func Parse(cmd ...string) (*Parsed, error) {
	args, err := cmdToArgs(cmd...)
	if err != nil {
		return nil, err
	}
	// Expand @file syntax in data parameters
	args, err = expandCurlDataFiles(args)
	if err != nil {
		return nil, err
	}

	out := newParsed()
	state := stateBlank

	for _, a := range args {
		switch {
		case isURL(a):
			u, err := url.Parse(a)
			if err != nil {
				return nil, err
			}
			out.URL = u
		case a == "-A" || a == "--user-agent":
			state = stateUA
		case a == "-H" || a == "--header":
			state = stateHeader
		case a == "-d" || a == "--data" || a == "--data-ascii" || a == "--data-raw":
			state = stateData
		case a == "-u" || a == "--user":
			state = stateUser
		case a == "-I" || a == "--head":
			out.Method = http.MethodHead
		case a == "-X" || a == "--request":
			state = stateMethod
		case a == "-b" || a == "--cookie":
			state = stateCookie
		case a == "--compressed":
			if out.Header.Get("Accept-Encoding") == "" {
				out.Header.Add("Accept-Encoding", "deflate, gzip")
			}
		case a != "":
			switch state {
			case stateHeader:
				k, v := parseField(a)
				out.Header.Add(k, v)
				state = stateBlank
			case stateUA:
				out.Header.Add("User-Agent", a)
				state = stateBlank
			case stateData:
				if out.Method == http.MethodGet || out.Method == http.MethodHead {
					out.Method = http.MethodPost
				}

				if len(out.Body) == 0 {
					out.Body = a
				} else {
					out.Body = out.Body + "&" + a
				}

				state = stateBlank
			case stateUser:
				out.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(a))))
				state = stateBlank
			case stateMethod:
				out.Method = a
				state = stateBlank
			case stateCookie:
				out.Header.Add("Cookie", a)
				state = stateBlank
			default:
			}
		}
	}

	if len(out.Body) > 0 && out.Header.Get("Content-Type") != "" {
		out.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	return out, nil
}

// Request returns *http.Request.
func (p *Parsed) Request() (*http.Request, error) {
	var b io.Reader
	if p.URL == nil {
		return nil, fmt.Errorf("curlreq: invalid URL: %s", p.URL)
	}
	if p.Body == "" {
		b = http.NoBody
	} else {
		b = strings.NewReader(p.Body)
	}
	req, err := http.NewRequest(p.Method, p.URL.String(), b)
	if err != nil {
		return nil, err
	}
	req.Header = p.Header
	return req, nil
}

func (p *Parsed) MarshalJSON() ([]byte, error) {
	s := struct {
		URL    string      `json:"url"`
		Method string      `json:"method"`
		Header http.Header `json:"header"`
		Body   string      `json:"body,omitempty"`
	}{
		URL:    p.URL.String(),
		Method: p.Method,
		Header: p.Header,
		Body:   p.Body,
	}
	return json.Marshal(s)
}

func newParsed() *Parsed {
	return &Parsed{
		Method: http.MethodGet,
		Header: http.Header{},
	}
}

func cmdToArgs(cmd ...string) ([]string, error) {
	var err error
	if len(cmd) == 1 {
		cmd, err = shellwords.Parse(cmd[0])
		if err != nil {
			return nil, err
		}
	}
	if cmd[0] != "curl" {
		return nil, fmt.Errorf("invalid curl command: %s", cmd)
	}
	if len(cmd) == 1 {
		return nil, fmt.Errorf("invalid curl command: %s", cmd)
	}

	return rewrite(cmd[1:]), nil
}

func rewrite(args []string) []string {
	rw := []string{}
	for _, a := range args {
		if strings.HasPrefix(a, "-X") {
			rw = append(rw, a[0:2])
			rw = append(rw, a[2:])
		} else {
			rw = append(rw, a)
		}
	}
	return rw
}

func isURL(u string) bool {
	return strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")
}

func parseField(a string) (string, string) {
	i := strings.Index(a, ":")
	return strings.TrimSpace(a[0:i]), strings.TrimSpace(a[i+1:])
}

// expandCurlDataFiles recognizes the @file syntax in data parameters and expands its content.
func expandCurlDataFiles(in []string) ([]string, error) {
	args := slices.Clone(in)
	for i := 0; i < len(args); {
		opt, value, inline, ok := parseCurlDataArg(args[i])
		if !ok {
			i++
			continue
		}

		if inline {
			step := 1
			if content, err := readDataFile(value); err != nil {
				return nil, err
			} else if content != "" {
				args[i] = opt
				args = slices.Insert(args, i+1, content)
				step = 2
			}
			i += step
			continue
		}

		if i+1 >= len(args) {
			break
		}

		if content, err := readDataFile(args[i+1]); err != nil {
			return nil, err
		} else if content != "" {
			args[i+1] = content
		}
		i += 2
	}

	return args, nil
}

// readDataFile reads the content of a file if the value starts with @, returns empty string otherwise.
func readDataFile(value string) (string, error) {
	if !strings.HasPrefix(value, "@") || len(value) <= 1 {
		return "", nil
	}
	payloadPath := value[1:]
	b, err := os.ReadFile(payloadPath)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", payloadPath, err)
	}
	return string(b), nil
}

func parseCurlDataArg(arg string) (option, value string, inline, ok bool) {
	switch {
	case arg == "--data-binary":
		return "--data-binary", "", false, true
	case strings.HasPrefix(arg, "--data-binary="):
		return "--data-binary", arg[len("--data-binary="):], true, true
	case arg == "--data-ascii":
		return "--data-ascii", "", false, true
	case strings.HasPrefix(arg, "--data-ascii="):
		return "--data-ascii", arg[len("--data-ascii="):], true, true
	case arg == "--data":
		return "--data", "", false, true
	case strings.HasPrefix(arg, "--data="):
		return "--data", arg[len("--data="):], true, true
	case arg == "-d":
		return "-d", "", false, true
	case strings.HasPrefix(arg, "-d"):
		return "-d", arg[len("-d"):], true, true
	default:
		return "", "", false, false
	}
}
