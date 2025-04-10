package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var _ Backend = (*httpBackend)(nil)

func NewHttpBackend(addr string) Backend {
	protocols := &http.Protocols{}
	protocols.SetUnencryptedHTTP2(true)

	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}

	return &httpBackend{
		addr: addr,
		client: &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: true,
				Protocols:         protocols,
			},
		},
	}
}

type httpBackend struct {
	addr   string
	client *http.Client
}

func (b *httpBackend) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", b.addr)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := b.client.Do(httpReq)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health request status not 200")
	}
	return nil
}

func (b *httpBackend) Invoke(ctx context.Context, req Request) (Response, error) {
	sb := strings.Builder{}
	sb.WriteString(b.addr)
	if req.Path[0] != '/' {
		sb.WriteRune('/')
	}
	sb.WriteString(req.Path)

	url := sb.String()
	br := bytes.NewReader(req.Body)

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, br)
	if err != nil {
		return Response{}, err
	}
	for k, v := range req.Headers {
		httpReq.Header.Add(k, v)
	}

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}

	headers := map[string]string{}
	headers["Content-Type"] = resp.Header.Get("Content-Type")

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	return Response{
		Status:  resp.StatusCode,
		Body:    body,
		Headers: headers,
	}, nil
}
