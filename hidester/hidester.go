// vim:set sw=2 sts=2:
package hidester

import (
  "bytes"
  "crypto/tls"
  "errors"
  "log"
  "io/ioutil"
  "net"
  "net/http"
  "net/http/cookiejar"
  neturl "net/url"
  "strconv"
  "strings"
  "time"
)

type Hidester struct {
  Jar *cookiejar.Jar
  LastURL string
  Conn HttpConnection
  ConnReady bool
  //
  Debug bool
}

func NewHidester() (*Hidester, error) {
  jar, err := cookiejar.New(nil)
  if err != nil {
    return nil, err
  }
  h := &Hidester{ Jar: jar, Debug: false }
  // start connection
  h.Conn = HttpConnection{}
  h.ConnReady = false
  go func() {
    ready := h.Conn.Open("us.hidester.com")
    <-ready
    h.ConnReady = true
  }()
  return h, nil
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

func (hs *Hidester)Get_old(url string) ([]byte, error) {
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

func (hs *Hidester)GetWithReferer_old(url, referer string) ([]byte, error) {
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

func (hs *Hidester)Get(url string) ([]byte, error) {
  for hs.ConnReady == false {
    time.Sleep(time.Millisecond * 100)
  }

  path := "/proxy.php?u=" + neturl.QueryEscape(url) + "&b=2"
  reqhdr := []string{"Referer: https://us.hidester.com/proxy.php"}
  body, err := hs.Conn.Get(path, reqhdr)
  if err != nil {
    // reopen
    hs.ConnReady = false
    go func() {
      ready := hs.Conn.Open("us.hidester.com")
      <-ready
      hs.ConnReady = true
    }()
    return nil, err
  }
  return body, nil
}

func (hs *Hidester)GetWithReferer(url, referer string) ([]byte, error) {
  for hs.ConnReady == false {
    time.Sleep(time.Millisecond * 100)
  }

  path := "/proxy.php?u=" + neturl.QueryEscape(url) + "&b=2"
  reqhdr := []string{"Referer: https://us.hidester.com/proxy.php?u=" + neturl.QueryEscape(referer) + "&b=2"}
  body, err := hs.Conn.Get(path, reqhdr)
  if err != nil {
    // reopen
    hs.ConnReady = false
    go func() {
      ready := hs.Conn.Open("us.hidester.com")
      <-ready
      hs.ConnReady = true
    }()
    return nil, err
  }
  return body, nil
}

type HttpConnection struct {
  State int
  host string
  conn net.Conn
  lasterr error
}

func (c *HttpConnection)Open(host string) chan bool {
  ready := make(chan bool)
  c.State = 0
  go func() {
    defer func() { ready <- true }()
    // https only
    conf := &tls.Config{ InsecureSkipVerify: true }
    conn, err := tls.Dial("tcp", host + ":443", conf)
    if err != nil {
      c.lasterr = err
      c.host = ""
      return
    }
    c.conn = conn
    c.host = host
    c.State = 1
  }()
  return ready
}

func (c *HttpConnection)Get(uri string, reqhdr []string) ([]byte, error) {
  // need check connection is ok
  if c.host == "" {
    return nil, errors.New("invalid connection")
  }
  req := "GET " + uri + " HTTP/1.1\r\n"
  req += strings.Join(reqhdr, "\r\n") + "\r\n"
  req += "Host: " + c.host + "\r\n"
  req += "Connection: Keep-Alive\r\n"
  req += "\r\n"
  //log.Println(req)
  if _, err := c.conn.Write([]byte(req)); err != nil {
    c.conn.Close()
    c.host = ""
    return nil, err
  }
  // get Response Header
  hdr := ""
  body := []byte{}
  for {
    buf := make([]byte, 1024)
    n, err := c.conn.Read(buf)
    if err != nil {
      // connection close?
      c.conn.Close()
      c.host = ""
      return nil, err
    }
    index := bytes.Index(buf[:n], []byte("\r\n\r\n"))
    if index >= 0 {
      hdr += string(buf[:index])
      body = buf[index+4:n]
      break
    }
    hdr += string(buf[:n])
  }
  //log.Println(hdr)
  // headers
  headers := strings.Split(hdr, "\r\n")
  length := 0
  for _, header := range(headers) {
    if strings.Index(header, "Location: ") == 0 {
      // 302
      u, err := neturl.Parse(header[10:])
      if err != nil {
	// bad location
	c.conn.Close()
	c.host = ""
	return nil, err
      }
      // assume no content
      path := u.Path
      if path[0] != '/' {
	path = "/" + path
      }
      return c.Get(path, reqhdr)
    }
    if strings.Index(header, "Content-Length: ") == 0 {
      length, _ = strconv.Atoi(header[16:])
      // ignore error
    }
    if strings.Index(header, "Transfer-Encoding: chunked") == 0 {
      // goto chunked mode
      body, err := c.Chunked()
      if err != nil {
	c.conn.Close()
	c.host = ""
	return nil, err
      }
      return body, nil
    }
  }
  for len(body) < length {
    buf := make([]byte, length)
    n, err := c.ReadWithTimeout(buf, 10 * time.Second)
    if err != nil {
      // connection close?
      c.conn.Close()
      c.host = ""
      return nil, err
    }
    body = append(body[:], buf[:n]...)
  }
  return body, nil
}

func (c *HttpConnection)Chunked() ([]byte, error) {
  body := []byte{}
  for {
    sz := 0
    for {
      buf := make([]byte, 1)
      n, err := c.conn.Read(buf)
      if n != 1 {
	if err != nil {
	  return nil, err
	}
	return nil, errors.New("size error")
      }
      if buf[0] == 13 { // \r
	c.conn.Read(buf) // \n
	break
      }
      sz *= 16
      if buf[0] <= '9' {
	sz += int(buf[0] - 0x30)
      } else if buf[0] <= 'F' {
	sz += int(buf[0] - 'A' + 10)
      } else if buf[0] <= 'f' {
	sz += int(buf[0] - 'a' + 10)
      }
    }
    if sz == 0 {
      dbuf := make([]byte, 2)
      c.conn.Read(dbuf)
      break
    }
    //log.Printf("chunk %d\n", sz)
    rd := 0
    for rd != sz {
      cbuf := make([]byte, sz)
      n, _ := c.conn.Read(cbuf)
      rd += n
      //log.Printf("read chunk %d/%d\n", rd, sz)
      body = append(body, cbuf[:n]...)
    }
    dbuf := make([]byte, 2)
    c.conn.Read(dbuf)
  }
  return body, nil
}

func (c *HttpConnection)ReadWithTimeout(b []byte, d time.Duration) (n int, err error) {
  // set default return
  n = 0
  err = nil
  ch := make(chan bool)
  go func() {
    n, err = c.conn.Read(b)
    ch <- true
  }()
  select {
  case <- ch: return
  case <- time.After(d): break
  }
  err = errors.New("read timeout")
  return
}
