package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/pkg/errors"
)

func (c *Client) runWithPostFields(ctx context.Context, req *Request, resp interface{}) error {
	operations, err := req.OperationsJSON()
	if err != nil {
		return err
	}

	params := map[string]string{
		"operations": string(operations),
		"map":        req.FileMap(),
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for i, file := range req.files {
		key := fmt.Sprintf("%d", i+1)

		part, err := writer.CreateFormFile(key, file.Name)
		if err != nil {
			return errors.Wrap(err, "creating form")
		}

		if _, err := io.Copy(part, file.R); err != nil {
			return errors.Wrap(err, "preparing file")
		}
	}

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	err = writer.Close()
	if err != nil {
		return errors.Wrap(err, "writing fields")
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	c.logf(">> operators: %s", operations)
	c.logf(">> files: %d", len(req.files))
	c.logf(">> query: %s", req.q)

	gr := &graphResponse{
		Data: resp,
	}

	r, err := http.NewRequest(http.MethodPost, c.endpoint, &body)
	if err != nil {
		return err
	}

	r.Close = c.closeReq
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("Accept", "application/json; charset=utf-8")

	for key, values := range req.Header {
		for _, value := range values {
			r.Header.Add(key, value)
		}
	}

	c.logf(">> headers: %v", r.Header)

	r = r.WithContext(ctx)

	res, err := c.httpClient.Do(r)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	var buf bytes.Buffer

	if _, err := io.Copy(&buf, res.Body); err != nil {
		return errors.Wrap(err, "reading body")
	}

	c.logf("<< %s", buf.String())

	if err := json.NewDecoder(&buf).Decode(&gr); err != nil {
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("graphql: server returned a non-200 status code: %v", res.StatusCode) //nolint:goerr113
		}

		return errors.Wrap(err, "decoding response")
	}

	if len(gr.Errors) > 0 {
		// return first error
		return gr.Errors[0]
	}

	return nil
}
