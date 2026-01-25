package request

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Requester struct {
	*http.Client
}

func NewRequester() *Requester {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	var client = &http.Client{
		Timeout:   time.Second * 30,
		Transport: transport,
	}
	return &Requester{client}
}

type Data struct {
	Method       string
	URL          string
	Header       map[string]string
	Body         io.Reader
	ExpectedCode int
	Bind         any
}

func (r *Requester) Request(data *Data) error {
	request, err := http.NewRequest(data.Method, data.URL, data.Body)
	if err != nil {
		return err
	}
	for key, value := range data.Header {
		request.Header.Set(key, value)
	}

	response, err := r.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != data.ExpectedCode {
		return fmt.Errorf("unexpected status code: %d, expected: %d", response.StatusCode, data.ExpectedCode)
	}
	content, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(content, data.Bind)
}
