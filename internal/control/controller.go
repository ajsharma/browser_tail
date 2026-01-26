// Package control provides browser automation capabilities for testing.
package control

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// Controller provides browser automation capabilities via CDP.
type Controller struct {
	allocatorCtx context.Context
	browserCtx   context.Context
	cancel       context.CancelFunc
	timeout      time.Duration
}

// NewController creates a new browser controller connected to the specified Chrome port.
func NewController(port string) (*Controller, error) {
	allocatorCtx, _ := chromedp.NewRemoteAllocator(context.Background(),
		"http://localhost:"+port)

	browserCtx, cancel := chromedp.NewContext(allocatorCtx)

	return &Controller{
		allocatorCtx: allocatorCtx,
		browserCtx:   browserCtx,
		cancel:       cancel,
		timeout:      30 * time.Second,
	}, nil
}

// SetTimeout sets the default timeout for operations.
func (c *Controller) SetTimeout(d time.Duration) {
	c.timeout = d
}

// Close releases resources.
func (c *Controller) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Navigate navigates to the specified URL.
func (c *Controller) Navigate(url string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Navigate(url))
}

// Click clicks on an element matching the selector.
func (c *Controller) Click(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Click(selector),
	)
}

// Type types text into an element matching the selector.
func (c *Controller) Type(selector, text string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Clear(selector),
		chromedp.SendKeys(selector, text),
	)
}

// Evaluate executes JavaScript and returns the result as JSON.
func (c *Controller) Evaluate(js string) (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var result interface{}
	err := chromedp.Run(ctx, chromedp.Evaluate(js, &result))
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// WaitVisible waits for an element to be visible.
func (c *Controller) WaitVisible(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.WaitVisible(selector))
}

// WaitReady waits for an element to be ready (present in DOM).
func (c *Controller) WaitReady(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.WaitReady(selector))
}

// Screenshot captures a screenshot and returns it as PNG bytes.
func (c *Controller) Screenshot() ([]byte, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf))
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// GetText retrieves the text content of an element.
func (c *Controller) GetText(selector string) (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var text string
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.Text(selector, &text),
	)
	if err != nil {
		return "", err
	}

	return text, nil
}

// GetAttribute retrieves an attribute value from an element.
func (c *Controller) GetAttribute(selector, attribute string) (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var value string
	var ok bool
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.AttributeValue(selector, attribute, &value, &ok),
	)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("attribute %q not found on %q", attribute, selector)
	}

	return value, nil
}

// GetHTML retrieves the outer HTML of an element.
func (c *Controller) GetHTML(selector string) (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.WaitVisible(selector),
		chromedp.OuterHTML(selector, &html),
	)
	if err != nil {
		return "", err
	}

	return html, nil
}

// ScrollTo scrolls to an element.
func (c *Controller) ScrollTo(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.ScrollIntoView(selector),
	)
}

// Focus focuses on an element.
func (c *Controller) Focus(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.Focus(selector),
	)
}

// Submit submits a form.
func (c *Controller) Submit(selector string) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx,
		chromedp.Submit(selector),
	)
}

// Sleep pauses execution for the specified duration.
func (c *Controller) Sleep(d time.Duration) error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Sleep(d))
}

// GetTitle returns the current page title.
func (c *Controller) GetTitle() (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var title string
	err := chromedp.Run(ctx, chromedp.Title(&title))
	if err != nil {
		return "", err
	}

	return title, nil
}

// GetURL returns the current page URL.
func (c *Controller) GetURL() (string, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var url string
	err := chromedp.Run(ctx, chromedp.Location(&url))
	if err != nil {
		return "", err
	}

	return url, nil
}

// GetNodes returns all nodes matching a selector.
func (c *Controller) GetNodes(selector string) ([]*cdp.Node, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var nodes []*cdp.Node
	err := chromedp.Run(ctx, chromedp.Nodes(selector, &nodes))
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// GetDocument returns the root DOM node.
func (c *Controller) GetDocument() (*cdp.Node, error) {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	var node *cdp.Node
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		doc, err := dom.GetDocument().Do(ctx)
		if err != nil {
			return err
		}
		node = doc
		return nil
	}))
	if err != nil {
		return nil, err
	}

	return node, nil
}

// Reload reloads the current page.
func (c *Controller) Reload() error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.Reload())
}

// Back navigates back in history.
func (c *Controller) Back() error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.NavigateBack())
}

// Forward navigates forward in history.
func (c *Controller) Forward() error {
	ctx, cancel := context.WithTimeout(c.browserCtx, c.timeout)
	defer cancel()

	return chromedp.Run(ctx, chromedp.NavigateForward())
}
