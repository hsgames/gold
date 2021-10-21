package http

import (
	"github.com/hsgames/gold/log"
	"io"
	"io/ioutil"
	"net/http"
	stdurl "net/url"
	"strings"
)

type Form = map[string]string

type Client struct {
	opts   clientOptions
	client *http.Client
	logger log.Logger
}

func NewClient(logger log.Logger, opt ...ClientOption) *Client {
	opts := defaultClientOptions()
	for _, o := range opt {
		o(&opts)
	}
	opts.ensure()
	return &Client{
		opts:   opts,
		client: &http.Client{Timeout: opts.timeout},
		logger: logger,
	}
}

func (c *Client) Get(url string, data Form) ([]byte, error) {
	if len(data) > 0 {
		values := stdurl.Values{}
		for k, v := range data {
			values.Set(k, v)
		}
		url += "?" + values.Encode()
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	bs, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (c *Client) Post(url string, contentType string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (c *Client) PostForm(url string, data Form) ([]byte, error) {
	values := stdurl.Values{}
	for k, v := range data {
		values.Set(k, v)
	}
	return c.Post(url, "application/x-www-form-urlencoded",
		strings.NewReader(values.Encode()))
}

func (c *Client) PostData(url string, data []byte) ([]byte, error) {
	return c.Post(url, "application/x-www-form-urlencoded",
		strings.NewReader(stdurl.QueryEscape(string(data))))
}
