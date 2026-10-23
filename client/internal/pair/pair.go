// Package pair makes this device a moon of the user's planet. It is the
// only part of the client that uses HTTP (eyre, logged in with +code);
// everything after it runs over mesa.
package pair

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"time"

	mesa "github.com/mopfel-winrux/libmesa/bindings/go"

	"github.com/mopfel-winrux/urbit-drive/client/internal/keys"
)

// Options says where the planet is and what to call this device.
type Options struct {
	URL       string // the planet's web address, e.g. https://sampel-palnet.tlon.network
	Code      string // +code
	Name      string // device name, shown on the planet
	Publisher string // ship to install %drive from if it isn't installed
	Fake      bool   // the planet is a fake ship (urbit -F): moon keys come from its name
	Log       func(string, ...any)
}

// Result is the paired moon.
type Result struct {
	Planet  mesa.Ship
	Moon    mesa.Ship
	Keyfile string // empty for fake ships
}

type eyre struct {
	base   string
	cookie string
	ship   string // without ~
	hc     *http.Client
}

// Pair logs in, installs %drive if needed, makes moon keys here and
// registers the moon with the planet.
func Pair(o Options) (*Result, error) {
	if o.Log == nil {
		o.Log = func(string, ...any) {}
	}
	e := &eyre{base: strings.TrimRight(o.URL, "/"), hc: &http.Client{Timeout: 30 * time.Second}}
	if err := e.login(o.Code); err != nil {
		return nil, err
	}
	planet, err := mesa.ParseShip("~" + e.ship)
	if err != nil {
		return nil, fmt.Errorf("login cookie names %q: %w", e.ship, err)
	}
	if planet.Rank() > mesa.Rank(2) {
		return nil, fmt.Errorf("~%s is a moon or comet; pair with a planet, star or galaxy", e.ship)
	}
	o.Log("logged in to ~%s", e.ship)

	if err := e.ensureDrive(o); err != nil {
		return nil, err
	}

	moon, err := keys.RandomMoon(planet)
	if err != nil {
		return nil, err
	}
	res := &Result{Planet: planet, Moon: moon}
	pass := "00"
	if !o.Fake {
		k, err := keys.New(moon)
		if err != nil {
			return nil, err
		}
		res.Keyfile, pass = k.Keyfile, k.PassHex()
	}
	o.Log("pairing as %s", moon)
	err = e.poke("drive", "json", map[string]any{
		"pair": map[string]string{"name": o.Name, "moon": moon.String(), "pass": pass},
	})
	if err != nil {
		return nil, err
	}
	for i := 0; i < 30; i++ {
		var r struct{ Paired bool }
		if err := e.scry("/drive/pair/"+moon.String()+".json", &r); err == nil && r.Paired {
			return res, nil
		}
		time.Sleep(time.Second)
	}
	return nil, errors.New("the planet didn't record the moon (see its dojo)")
}

func (e *eyre) login(code string) error {
	resp, err := e.hc.PostForm(e.base+"/~/login", url.Values{"password": {code}})
	if err != nil {
		return err
	}
	resp.Body.Close()
	for _, c := range resp.Cookies() {
		if s, ok := strings.CutPrefix(c.Name, "urbauth-~"); ok {
			e.cookie, e.ship = c.Name+"="+c.Value, s
			return nil
		}
	}
	return fmt.Errorf("login failed (%s): wrong +code?", resp.Status)
}

func (e *eyre) do(req *http.Request) (*http.Response, error) {
	req.Header.Set("Cookie", e.cookie)
	return e.hc.Do(req)
}

var errNoScry = errors.New("no scry result")

func (e *eyre) scry(path string, out any) error {
	req, _ := http.NewRequest("GET", e.base+"/~/scry"+path, nil)
	resp, err := e.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return errNoScry
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("scry %s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// poke sends one poke on a fresh channel and waits for its ack.
func (e *eyre) poke(app, mark string, body any) error {
	ch := fmt.Sprintf("%s/~/channel/drive-pair-%d-%d", e.base, time.Now().Unix(), rand.Uint32())
	msg, _ := json.Marshal([]map[string]any{{
		"id": 1, "action": "poke", "ship": e.ship, "app": app, "mark": mark, "json": body,
	}})
	req, _ := http.NewRequest("PUT", ch, bytes.NewReader(msg))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("poke %%%s: %s", app, resp.Status)
	}
	defer func() {
		req, _ := http.NewRequest("DELETE", ch, nil)
		if r, err := e.do(req); err == nil {
			r.Body.Close()
		}
	}()
	// the ack comes on the channel's event stream
	req, _ = http.NewRequest("GET", ch, nil)
	req.Header.Set("Accept", "text/event-stream")
	resp, err = e.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		for _, line := range strings.Split(string(buf), "\n") {
			data, ok := strings.CutPrefix(line, "data: ")
			if !ok {
				continue
			}
			var ev struct {
				Response string
				Ok       *string `json:"ok"`
				Err      *string `json:"err"`
			}
			if json.Unmarshal([]byte(data), &ev) != nil || ev.Response != "poke" {
				continue
			}
			if ev.Err != nil {
				return fmt.Errorf("poke %%%s nacked: %s", app, *ev.Err)
			}
			return nil
		}
		if err == io.EOF {
			return fmt.Errorf("poke %%%s: channel closed before the ack", app)
		}
		if err != nil {
			return err
		}
	}
}

// ensureDrive installs %drive from the publisher if it isn't running,
// and waits for it to answer.
func (e *eyre) ensureDrive(o Options) error {
	var v struct{ Version string }
	if e.scry("/drive/version.json", &v) == nil {
		o.Log("%%drive %s is installed", v.Version)
		return nil
	}
	if o.Publisher == "" {
		return errors.New("%drive isn't installed on the planet, and no publisher is configured")
	}
	o.Log("installing %%drive from %s", o.Publisher)
	err := e.poke("hood", "kiln-install", map[string]string{
		"local": "drive", "ship": o.Publisher, "desk": "drive",
	})
	if err != nil {
		return err
	}
	for i := 0; i < 120; i++ {
		time.Sleep(5 * time.Second)
		if e.scry("/drive/version.json", &v) == nil {
			o.Log("%%drive %s installed", v.Version)
			return nil
		}
	}
	return errors.New("%drive didn't start within 10 minutes of installing")
}
