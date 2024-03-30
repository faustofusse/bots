package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"unicode"

	_ "github.com/joho/godotenv/autoload"
)

var client http.Client

var userAgent string = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"

var hlsHeaders map[string]string = map[string]string{
    "Accept": "*/*",
    "Accept-Encoding": "gzip, deflate, br, zstd",
    "Accept-Language": "en-US,en;q=0.9",
    "Cache-Control": "no-cache",
    "Origin": "https://frontendmasters.com",
    "Pragma": "no-cache",
    "Referer": "https://frontendmasters.com/",
    "Sec-Ch-Ua": "\"Brave\";v=\"123\", \"Not:A-Brand\";v=\"8\", \"Chromium\";v=\"123\"",
    "Sec-Ch-Ua-Mobile": "?0",
    "Sec-Ch-Ua-Platform": "\"macOS\"",
    "Sec-Fetch-Dest": "empty",
    "Sec-Fetch-Mode": "cors",
    "Sec-Fetch-Site": "same-site",
    "Sec-Gpc": "1",
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
}

var headers map[string]string = map[string]string{
  "cookie": os.Getenv("COOKIE"),
  "accept": "application/json", 
  "accept-language": "en-US,en;q=0.9",
  "content-type": "application/json; charset=utf-8",
  "origin": "https://frontendmasters.com",
  "referer": "https://frontendmasters.com/",
  "sec-ch-ua": "\"Brave\";v=\"123\", \"Not:A-Brand\";v=\"8\", \"Chromium\";v=\"123\"",
  "sec-ch-ua-mobile": "?0",
  "sec-ch-ua-platform": "macOS",
  "sec-fetch-dest": "empty",
  "sec-fetch-mode": "cors",
  "sec-fetch-site": "same-site",
  "sec-gpc": "1",
  "user-agent": userAgent,
}

func capitalizeHeader(header string) string {
    result := ""
    for i, v := range header {
        if i == 0 || (i - 1 > 0 && header[i - 1] == '-') {
            result += string(unicode.ToUpper(v))
        } else {
            result += string(v)
        }
    }
    return result
}

func request(method string, url string, body any, result any) ([]*http.Cookie, error) {
    marshaled, err := json.Marshal(body)
    if err != nil { return nil, err }
    req, err := http.NewRequest(method, url, bytes.NewBuffer(marshaled))
    if err != nil { return nil, err }
    for key, value := range headers {
        req.Header.Add(key, value)
    }
    res, err := client.Do(req)
    if err != nil { return nil, err }
    defer res.Body.Close()
    responseBody, err := io.ReadAll(res.Body)
    if err != nil { return nil, err }
    return append(req.Cookies(), res.Cookies()...), json.Unmarshal(responseBody, result)
}

func execute(cmd *exec.Cmd, title string) {
    fmt.Print("\n")
    stderr, _ := cmd.StderrPipe()
    cmd.Start()
    scanner := bufio.NewScanner(stderr)
    scanner.Split(bufio.ScanWords)
    for scanner.Scan() {
        m := scanner.Text()
        if strings.ContainsRune(m, '%') {
            percentage := strings.Split(m, "%")[0]
            fmt.Printf("\033[1ADownloading lesson: %v - %v%% \n", title, percentage)
        }
    }
    cmd.Wait()
}

func filterCookies(cookies []*http.Cookie, lessonHash string) []*http.Cookie {
    filtered := []*http.Cookie{}
    for _, cookie := range cookies {
        if !strings.Contains(cookie.Path, lessonHash) {
            filtered = append(filtered, cookie)
        }
    }
    return filtered
}

func download(lesson map[string]any) {
    hash := lesson["hash"].(string)
    url := "https://api.frontendmasters.com/v1/kabuki/video/" + hash + "/source?f=m3u8"
    response := map[string]any{}
    cookies, err := request("GET", url, nil, &response)
    if err != nil { log.Fatal(err.Error()) }
    hls := response["url"].(string)
    filename := lesson["slug"].(string) + ".mp4"
    args := []string{}
    args = append(args, "-m", "ffpb")
    headers := ""
    for key, value := range hlsHeaders {
        headers += capitalizeHeader(key) + ":" + value + "\r\n"
    }
    header := "Cookie: "
    for _, cookie := range filterCookies(cookies, hash) {
        header += cookie.Name + "=" + cookie.Value + "; "
    }
    headers += header + "\r\n"
    args = append(args, "-headers", "'" + headers + "'")
    args = append(args, "-user_agent", "'" + userAgent + "'")
    args = append(args, "-f", "hls")
    args = append(args, "-i", hls)
    args = append(args, "-c", "copy", "-bsf:a", "aac_adtstoasc")
    args = append(args, filename)
    cmd := exec.Command("python3", args...)
    execute(cmd, lesson["title"].(string))
}

func main() {
    client = http.Client{}
    course := os.Args[1]
    url := "https://api.frontendmasters.com/v1/kabuki/courses/" + course
    response := map[string]any{}
    _, err := request("GET", url, nil, &response)
    if err != nil { log.Fatal(err.Error()) }
    lessonsMap := response["lessonData"].(map[string]any)
    lessons := make([]map[string]any, len(lessonsMap))
    for hash, value := range lessonsMap {
        lesson := value.(map[string]any)
        index := int(lesson["index"].(float64))
        lesson["hash"] = hash
        lessons[index] = lesson
    }
    for _, lesson := range lessons {
        download(lesson)
    }
}
