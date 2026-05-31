package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func DoJSON(
	ctx context.Context,
	client *http.Client,
	method, url string,
	headers map[string]string,
	in, out any,
	statusOK ...int,
) (status int, body []byte, err error) {

	var reqBody io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return 0, nil, fmt.Errorf("httpx: marshal: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	ok := false
	if len(statusOK) == 0 {
		ok = resp.StatusCode >= 200 && resp.StatusCode < 300
	} else {
		for _, c := range statusOK {
			if c == resp.StatusCode {
				ok = true
				break
			}
		}
	}
	if !ok {
		return resp.StatusCode, body, fmt.Errorf("httpx: bad status %d", resp.StatusCode)
	}

	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return resp.StatusCode, body, fmt.Errorf("httpx: decode: %w", err)
		}
	}
	return resp.StatusCode, body, nil
}
