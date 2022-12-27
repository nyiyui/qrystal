// The MIT License (MIT)
//
// Copyright (c) 2014 Cenk Altı
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

// Package rpc3 provides utilities for the github.com/cenkalti/rpc2 package.
//
// NOTE: it seemed like the gob-encoding did not work for central.CC, but JSON
// encoding does work (config), so we're going to use that instead.
package rpc3

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"

	"github.com/cenkalti/rpc2"
)

type jsonCodec struct {
	rwc   io.ReadWriteCloser
	dec   *json.Decoder
	enc   *json.Encoder
	mutex sync.Mutex
}

type message struct {
	Seq    uint64
	Method string
	Error  string
}

// NewJSONCodec returns a new rpc2.Codec using JSON encoding/decoding on conn.
func NewJSONCodec(conn io.ReadWriteCloser) rpc2.Codec {
	buf := bufio.NewWriter(conn)
	return &jsonCodec{
		rwc: conn,
		dec: json.NewDecoder(conn),
		enc: json.NewEncoder(buf),
	}
}

func (c *jsonCodec) ReadHeader(req *rpc2.Request, resp *rpc2.Response) error {
	var msg message
	if err := c.dec.Decode(&msg); err != nil {
		return err
	}

	if msg.Method != "" {
		req.Seq = msg.Seq
		req.Method = msg.Method
	} else {
		resp.Seq = msg.Seq
		resp.Error = msg.Error
	}
	return nil
}

func (c *jsonCodec) ReadRequestBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *jsonCodec) ReadResponseBody(body interface{}) error {
	return c.dec.Decode(body)
}

func (c *jsonCodec) WriteRequest(r *rpc2.Request, body interface{}) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err = c.enc.Encode(r); err != nil {
		return
	}
	if err = c.enc.Encode(body); err != nil {
		return
	}
	return nil
}

func (c *jsonCodec) WriteResponse(r *rpc2.Response, body interface{}) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err = c.enc.Encode(r); err != nil {
		return
	}
	if err = c.enc.Encode(body); err != nil {
		return
	}
	return nil
}

func (c *jsonCodec) Close() error {
	return c.rwc.Close()
}
