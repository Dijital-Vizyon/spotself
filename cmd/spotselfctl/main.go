package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

func main() {
	baseURL := flag.String("url", env("SPOTSELF_URL", "http://localhost:8080"), "SpotSelf server URL")
	token := flag.String("token", env("SPOTSELF_ADMIN_TOKEN", ""), "admin bearer token")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	c := client{
		baseURL: strings.TrimRight(*baseURL, "/"),
		token:   *token,
		http:    &http.Client{Timeout: 2 * time.Minute},
	}
	if err := run(c, flag.Arg(0), flag.Args()[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "spotselfctl:", err)
		os.Exit(1)
	}
}

func run(c client, command string, args []string) error {
	switch command {
	case "events":
		return c.requestJSON(http.MethodGet, "/api/events", nil, os.Stdout)
	case "stats":
		return c.requestJSON(http.MethodGet, "/api/stats", nil, os.Stdout)
	case "health":
		return c.requestJSON(http.MethodGet, "/api/health", nil, os.Stdout)
	case "create-event":
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		name := fs.String("name", "", "event name")
		watermark := fs.String("watermark", "", "watermark text")
		retention := fs.Int("retention", 30, "retention days")
		_ = fs.Parse(args)
		if strings.TrimSpace(*name) == "" {
			return fmt.Errorf("-name is required")
		}
		body := map[string]any{"name": *name, "watermark": *watermark, "retentionDays": *retention}
		return c.requestJSON(http.MethodPost, "/api/events", body, os.Stdout)
	case "upload":
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		event := fs.String("event", "", "event id or slug")
		_ = fs.Parse(args)
		if *event == "" || fs.NArg() == 0 {
			return fmt.Errorf("usage: spotselfctl upload -event <id> <file...>")
		}
		return c.upload(*event, "photos", fs.Args(), "/api/events/"+*event+"/photos")
	case "match":
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		event := fs.String("event", "", "event id or slug")
		threshold := fs.String("threshold", "0.56", "match threshold")
		_ = fs.Parse(args)
		if *event == "" || fs.NArg() != 1 {
			return fmt.Errorf("usage: spotselfctl match -event <id> [-threshold 0.56] <selfie>")
		}
		return c.uploadWithFields(*event, "selfie", fs.Args(), "/api/events/"+*event+"/match", map[string]string{"threshold": *threshold})
	case "photos":
		fs := flag.NewFlagSet(command, flag.ExitOnError)
		event := fs.String("event", "", "event id or slug")
		_ = fs.Parse(args)
		if *event == "" {
			return fmt.Errorf("-event is required")
		}
		return c.requestJSON(http.MethodGet, "/api/events/"+*event+"/photos", nil, os.Stdout)
	case "delete-event":
		if len(args) != 1 {
			return fmt.Errorf("usage: spotselfctl delete-event <event-id>")
		}
		return c.requestNoBody(http.MethodDelete, "/api/events/"+args[0])
	case "delete-photo":
		if len(args) != 2 {
			return fmt.Errorf("usage: spotselfctl delete-photo <event-id> <photo-id-or-file>")
		}
		return c.requestNoBody(http.MethodDelete, "/api/events/"+args[0]+"/photos/"+args[1])
	case "purge":
		return c.requestJSON(http.MethodPost, "/api/maintenance/purge", map[string]any{}, os.Stdout)
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func (c client) requestJSON(method, path string, body any, out io.Writer) error {
	var payload io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
		payload = &buf
	}
	req, err := http.NewRequest(method, c.baseURL+path, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, mustRead(resp.Body), "", "  "); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, pretty.String())
	return err
}

func (c client) requestNoBody(method, path string) error {
	req, err := http.NewRequest(method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	return nil
}

func (c client) upload(eventID, field string, files []string, path string) error {
	return c.uploadWithFields(eventID, field, files, path, nil)
}

func (c client) uploadWithFields(_ string, field string, files []string, path string, fields map[string]string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return err
		}
	}
	for _, filePath := range files {
		if err := addFile(writer, field, filePath); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.authorize(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	_, err = io.Copy(os.Stdout, resp.Body)
	if err == nil {
		fmt.Println()
	}
	return err
}

func addFile(writer *multipart.Writer, field, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	part, err := writer.CreateFormFile(field, filepath.Base(filePath))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func mustRead(r io.Reader) []byte {
	data, err := io.ReadAll(r)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func (c client) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `Usage:
  spotselfctl [flags] events
  spotselfctl [flags] stats
  spotselfctl [flags] health
  spotselfctl [flags] create-event -name "Event" [-watermark "..."] [-retention 30]
  spotselfctl [flags] upload -event <id> <file...>
  spotselfctl [flags] match -event <id> [-threshold 0.56] <selfie>
  spotselfctl [flags] photos -event <id>
  spotselfctl [flags] delete-event <event-id>
  spotselfctl [flags] delete-photo <event-id> <photo-id-or-file>
  spotselfctl [flags] purge

Flags:
`)
	flag.PrintDefaults()
}
