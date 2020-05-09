// vim:set sw=2 sts=2:
package hidester

import (
  "log"
  "io/ioutil"
  "net/http"
  "net/http/cookiejar"
  neturl "net/url"
  //"strings"
)

type Hidester struct {
  Jar *cookiejar.Jar
  LastURL string
  //
  Debug bool
}

func NewHidester() (*Hidester, error) {
  jar, err := cookiejar.New(nil)
  if err != nil {
    return nil, err
  }
  return &Hidester{ Jar: jar, Debug: false }, nil
}

func (hs *Hidester)dbgln(a ...interface{}) {
  if hs.Debug {
    log.Println(a...)
  }
}

func (hs *Hidester)dbgf(f string, a ...interface{}) {
  if hs.Debug {
    log.Printf(f, a...)
  }
}

func (hs *Hidester)Get(url string) ([]byte, error) {
  cl := &http.Client{ Jar: hs.Jar }

  query := "https://us.hidester.com/proxy.php?u=" + neturl.QueryEscape(url) + "&b=2"
  req, err := http.NewRequest("GET", query, nil)
  req.Header.Add("Referer", "https://us.hidester.com/proxy.php")
  resp, err := cl.Do(req)
  if err != nil {
    return nil, err
  }
  hs.dbgln(resp)
  defer resp.Body.Close()
  body, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }
  return body, nil
}

func (hs *Hidester)GetWithReferer(url, referer string) ([]byte, error) {
  cl := &http.Client{ Jar: hs.Jar }

  query := "https://us.hidester.com/proxy.php?u=" + neturl.QueryEscape(url) + "&b=2"
  req, err := http.NewRequest("GET", query, nil)
  req.Header.Add("Referer", "https://us.hidester.com/proxy.php?u=" + neturl.QueryEscape(referer) + "&b=2")
  resp, err := cl.Do(req)
  if err != nil {
    return nil, err
  }
  hs.dbgln(resp)
  body, err := ioutil.ReadAll(resp.Body)
  defer resp.Body.Close()
  if err != nil {
    return nil, err
  }
  return body, nil
}
