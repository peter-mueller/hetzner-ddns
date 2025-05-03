package hetzner

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	AuthAPIToken string
}

func (c *Client) do(method string, path string, body io.Reader) (*http.Response, error) {
	base, err := url.Parse("https://dns.hetzner.com/")
	if err != nil {
		return nil, err
	}
	u := base.ResolveReference(&url.URL{
		Path: path,
	})

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Auth-API-Token", c.AuthAPIToken)

	return http.DefaultClient.Do(req)
}

type Record struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	ZoneId string `json:"zone_id"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	TTL    int    `json:"ttl"`
}

func (c *Client) GetAllRecords() (records []Record, err error) {
	r, err := c.do("GET", "/api/v1/records", nil)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf(
			"unexpected status code %d: %s",
			r.StatusCode,
			r.Status,
		)
	}

	response := struct {
		Records []Record `json:"records"`
	}{}

	err = json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.Records, nil
}

func (c *Client) UpdateRecord(record Record) error {
	if record.Id == "" {
		return errors.New("id darf nicht leer sein")
	}
	buffer := new(bytes.Buffer)
	err := json.NewEncoder(buffer).Encode(record)
	if err != nil {
		return err
	}

	r, err := c.do("PUT", "/api/v1/records/"+record.Id, buffer)
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return fmt.Errorf(
			"unexpected status code %d: %s",
			r.StatusCode,
			r.Status,
		)
	}
	return nil
}
