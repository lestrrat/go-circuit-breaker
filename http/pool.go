package http

import (
	"sync"

	"github.com/pkg/errors"
)

var doCtxPool = sync.Pool{New: allocDoCtx}

// return a doCtx type
func getDoCtx() *doCtx {
	return doCtxPool.Get().(*doCtx)
}

func allocDoCtx() interface{} {
	return &doCtx{}
}

func releaseDoCtx(c *doCtx) {
	c.Error = nil
	c.ErrorOnBadStatus = false
	c.Request = nil
	c.Response = nil
	doCtxPool.Put(c)
}

// Execute fulfills the Circuit interface
func (c *doCtx) Execute() error {
	c.Response, c.Error = c.Client.Do(c.Request)
	if c.ErrorOnBadStatus && c.Response.StatusCode > 499 {
		c.Error = errors.Wrapf(ErrBadStatus, "received bad status %d", c.Response.StatusCode)
	}
	return c.Error
}

var getCtxPool = sync.Pool{New: allocGetCtx}

// return a getCtx type
func getGetCtx() *getCtx {
	return getCtxPool.Get().(*getCtx)
}

func allocGetCtx() interface{} {
	return &getCtx{}
}

func releaseGetCtx(c *getCtx) {
	c.Error = nil
	c.ErrorOnBadStatus = false
	c.URL = ""
	c.Response = nil
	getCtxPool.Put(c)
}

// Execute fulfills the Circuit interface
func (c *getCtx) Execute() error {
	c.Response, c.Error = c.Client.Get(c.URL)
	if c.ErrorOnBadStatus && c.Response.StatusCode > 499 {
		c.Error = errors.Wrapf(ErrBadStatus, "received bad status %d", c.Response.StatusCode)
	}
	return c.Error
}

var headCtxPool = sync.Pool{New: allocHeadCtx}

// return a headCtx type
func getHeadCtx() *headCtx {
	return headCtxPool.Get().(*headCtx)
}

func allocHeadCtx() interface{} {
	return &headCtx{}
}

func releaseHeadCtx(c *headCtx) {
	c.Error = nil
	c.ErrorOnBadStatus = false
	c.URL = ""
	c.Response = nil
	headCtxPool.Put(c)
}

// Execute fulfills the Circuit interface
func (c *headCtx) Execute() error {
	c.Response, c.Error = c.Client.Head(c.URL)
	if c.ErrorOnBadStatus && c.Response.StatusCode > 499 {
		c.Error = errors.Wrapf(ErrBadStatus, "received bad status %d", c.Response.StatusCode)
	}
	return c.Error
}

var postCtxPool = sync.Pool{New: allocPostCtx}

// return a postCtx type
func getPostCtx() *postCtx {
	return postCtxPool.Get().(*postCtx)
}

func allocPostCtx() interface{} {
	return &postCtx{}
}

func releasePostCtx(c *postCtx) {
	c.Body = nil
	c.BodyType = ""
	c.Error = nil
	c.ErrorOnBadStatus = false
	c.URL = ""
	c.Response = nil
	postCtxPool.Put(c)
}

// Execute fulfills the Circuit interface
func (c *postCtx) Execute() error {
	c.Response, c.Error = c.Client.Post(c.URL, c.BodyType, c.Body)
	if c.ErrorOnBadStatus && c.Response.StatusCode > 499 {
		c.Error = errors.Wrapf(ErrBadStatus, "received bad status %d", c.Response.StatusCode)
	}
	return c.Error
}

var postFormCtxPool = sync.Pool{New: allocPostFormCtx}

// return a postFormCtx type
func getPostFormCtx() *postFormCtx {
	return postFormCtxPool.Get().(*postFormCtx)
}

func allocPostFormCtx() interface{} {
	return &postFormCtx{}
}

func releasePostFormCtx(c *postFormCtx) {
	c.Error = nil
	c.ErrorOnBadStatus = false
	c.URL = ""
	c.Response = nil
	postFormCtxPool.Put(c)
}

// Execute fulfills the Circuit interface
func (c *postFormCtx) Execute() error {
	c.Response, c.Error = c.Client.PostForm(c.URL, c.Data)
	if c.ErrorOnBadStatus && c.Response.StatusCode > 499 {
		c.Error = errors.Wrapf(ErrBadStatus, "received bad status %d", c.Response.StatusCode)
	}
	return c.Error
}
