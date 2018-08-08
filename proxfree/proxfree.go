// vim:set sw=2 sts=2:
package proxfree

import (
  "log"
  "fmt"
  "io/ioutil"
  neturl "net/url"
  "crypto/tls"
  "strings"
  "strconv"
  "errors"
)

type ProxFree struct {
  Token string
  Session string
  Server string
  UA string
  //
  Debug bool
  // internal
  conn *tls.Conn
}

func NewProxFree() (*ProxFree, error) {
  token := getToken()
  if token == "" {
    return nil, errors.New("no token")
  }
  mozilla := "Mozilla/5.0 (Windows NT 6.1; WOW64) "
  webkit := "AppleWebKit/537.36 (KHTML, like Gecko) "
  chrome := "Chrome/51.0.2704.103 Safari/537.36"
  return &ProxFree{
    Token: token,
    Server: "ca.proxfree.com",
    UA: mozilla + webkit + chrome,
    Debug: false,
    conn: nil }, nil
}

func (pf *ProxFree)dbgln(a ...interface{}) {
  if pf.Debug {
    log.Println(a...)
  }
}

func (pf *ProxFree)dbgf(f string, a ...interface{}) {
  if pf.Debug {
    log.Printf(f, a...)
  }
}

func (pf *ProxFree)getSession() {
  conn, err := https(pf.Server + ":443")
  if err != nil {
    return
  }
  req := strings.Join([]string{
      "POST /request.php?do=go HTTP/1.1",
      "Host: " + pf.Server,
      "User-Agent: " + pf.UA,
      "Cookie: token=" + pf.Token,
      "Connection: close",
    }, "\r\n") + "\r\n\r\n"
  pf.dbgln(req)
  conn.Write([]byte(req))
  buf := make([]byte, 1024)
  n, err := conn.Read(buf)
  if err != nil {
    return
  }
  pf.dbgln(string(buf[:n]))
  s := findSetCookie(string(buf[:n]), "s")
  pf.Session = s
}

func (pf *ProxFree)Close() {
  if pf.conn != nil {
    pf.conn.Close()
    pf.conn = nil
  }
}

func (pf *ProxFree)readChunked(conn *tls.Conn) ([]byte, error) {
  // must start chunk size
  body := []byte{}
  for {
    sz := 0
    for {
      buf := make([]byte, 1)
      n, err := conn.Read(buf)
      if n != 1 {
	if err != nil {
	  return nil, err
	}
	return nil, errors.New("size error")
      }
      if buf[0] == 13 { // \r
	conn.Read(buf) // \n
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
      conn.Read(dbuf)
      break
    }
    pf.dbgf("chunk %d\n", sz)
    rd := 0
    for rd != sz {
      cbuf := make([]byte, sz)
      n, _ := conn.Read(cbuf)
      rd += n
      pf.dbgf("read chunk %d/%d\n", rd, sz)
      body = append(body, cbuf[:n]...)
    }
    dbuf := make([]byte, 2)
    conn.Read(dbuf)
  }
  return body, nil
}

func (pf *ProxFree)GetPage(page string) ([]byte, error) {
  conn := pf.conn
  if conn == nil {
    if pf.Session == "" {
      pf.getSession()
    }
    pf.dbgf("open https\n")
    conn, err := https(pf.Server + ":443")
    if err != nil {
      return nil, errors.New("HTTPS_ERROR")
    }
    pf.conn = conn
  }

  req := strings.Join([]string{
      "GET " + page + " HTTP/1.1",
      "Host: " + pf.Server,
      "User-Agent: " + pf.UA,
      "Cookie: token=" + pf.Token + "; s=" + pf.Session,
      "Connection: Keep-Alive",
    }, "\r\n") + "\r\n\r\n"
  pf.dbgln(req)
  conn.Write([]byte(req))
  buf := make([]byte, 1024)
  n, err := conn.Read(buf)
  if err != nil {
    conn.Close()
    pf.conn = nil
    return nil, errors.New("GET PAGE ERROR")
  }
  pf.dbgf("n=%d\n", n)
  pf.dbgln(string(buf[:n]))

  // check content and length
  hdr := string(buf[:n])
  if findTransferEncoding(hdr) == "chunked" {
    //Transfer-Encoding: chunked
    return pf.readChunked(conn)
  }
  ctype := findContentType(hdr)
  length := findContentLength(hdr)
  pf.dbgf("type: %s, length: %d\n", ctype, length)

  // already read a part?
  m := n
  for i := 3; i < n; i++ {
    if buf[i-3] == 13 && buf[i-2] == 10 && buf[i-1] == 13 && buf[i] == 10 {
      m = i + 1
      pf.dbgf("%d bytes\n", n - m)
      break
    }
  }

  body, _ := ioutil.ReadAll(conn)
  pf.dbgf("get %d bytes\n", len(body))
  if len(body) != length {
    conn.Close()
    pf.conn = nil
    return nil, errors.New("length error")
  }

  return body, nil
}

func (pf *ProxFree)Get(url string) ([]byte, error) {
  if pf.Session == "" {
    pf.getSession()
  }
  // TODO check pf.conn?
  pf.dbgf("open https\n")
  conn, err := https(pf.Server + ":443")
  if err != nil {
    return nil, errors.New("HTTPS ERROR")
  }
  pf.conn = conn

  // param
  values := neturl.Values{}
  values.Set("get", url)
  values.Set("allowCookies", "on")
  values.Set("pfipDropdown", "default")
  values.Set("pfserverDropdown", "https://" + pf.Server + "/request.php?do=go")
  content := values.Encode()

  req := strings.Join([]string{
      "POST /request.php?do=go HTTP/1.1",
      "Host: " + pf.Server,
      "User-Agent: " + pf.UA,
      "Cookie: token=" + pf.Token + "; s=" + pf.Session,
      "Connection: Keep-Alive",
      "Content-Length: " + fmt.Sprintf("%d", len(content)),
      "Content-Type: application/x-www-form-urlencoded",
    }, "\r\n") + "\r\n\r\n" + content
  pf.dbgln(req)
  conn.Write([]byte(req))
  buf := make([]byte, 1024)
  n, err := conn.Read(buf)
  if err != nil {
    conn.Close()
    pf.conn = nil
    return nil, errors.New("POST ERROR")
  }
  pf.dbgln(string(buf[:n]))
  // there must be Location:
  loc := findLocation(string(buf[:n]))
  if loc == "" {
    conn.Close()
    pf.conn = nil
    return nil, errors.New("NO Location")
  }
  rloc := "/page.php" + strings.SplitN(loc, "/page.php", 2)[1]
  pf.dbgln(rloc)

  return pf.GetPage(rloc)
}

//
func getToken() string {
  host := "www.proxfree.com"
  conn, err := https(host + ":443")
  if err != nil {
    return ""
  }
  defer conn.Close()

  req := strings.Join([]string{
      "GET / HTTP/1.1",
      "Host: " + host,
      "Connection: close",
    }, "\r\n") + "\r\n\r\n"
  conn.Write([]byte(req))

  buf := make([]byte, 1024)
  n, err := conn.Read(buf)
  if err != nil {
    return ""
  }

  return findSetCookie(string(buf[:n]), "token")
}

func https(url string) (*tls.Conn, error) {
  conf := &tls.Config { InsecureSkipVerify: true }
  conn, err := tls.Dial("tcp", url, conf)
  return conn, err
}

func findSetCookie(header string, key string) string {
  resp := strings.Split(header, "\r\n")
  for _, line := range(resp) {
    kv := strings.SplitN(line, " ", 2)
    if kv[0] == "Set-Cookie:" {
      cks := strings.Split(kv[1], "; ")
      for _, ck := range(cks) {
	a := strings.Split(ck, "=")
	if a[0] == key {
	  return a[1]
	}
      }
    }
  }
  return ""
}

func findLocation(header string) string {
  resp := strings.Split(header, "\r\n")
  for _, line := range(resp) {
    kv := strings.SplitN(line, " ", 2)
    if kv[0] == "Location:" {
      return kv[1]
    }
  }
  return ""
}

func findContentType(header string) string {
  resp := strings.Split(header, "\r\n")
  for _, line := range(resp) {
    kv := strings.SplitN(line, " ", 2)
    if kv[0] == "Content-Type:" {
      return kv[1]
    }
  }
  return ""
}

func findTransferEncoding(header string) string {
  resp := strings.Split(header, "\r\n")
  for _, line := range(resp) {
    kv := strings.SplitN(line, " ", 2)
    if kv[0] == "Transfer-Encoding:" {
      return kv[1]
    }
  }
  return ""
}

func findContentLength(header string) int {
  resp := strings.Split(header, "\r\n")
  for _, line := range(resp) {
    kv := strings.SplitN(line, " ", 2)
    if kv[0] == "Content-Length:" {
      l, _ := strconv.Atoi(kv[1])
      return l
    }
  }
  return 0
}
