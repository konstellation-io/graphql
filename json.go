package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func (c *Client) runWithJSON(ctx context.Context, req *Request, resp interface{}) error {
	var requestBody bytes.Buffer

	requestBodyObj := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     req.q,
		Variables: req.vars,
	}

	if err := json.NewEncoder(&requestBody).Encode(requestBodyObj); err != nil {
		return errors.Wrap(err, "encode body")
	}

	c.logf(">> variables: %v", req.vars)
	c.logf(">> query: %s", req.q)

	gr := &graphResponse{
		Data: resp,
	}

	r, err := http.NewRequest(http.MethodPost, c.endpoint, &requestBody)

	if err != nil {
		return err
	}

	r.Close = c.closeReq
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
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
