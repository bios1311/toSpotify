package apis

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"gopkg.in/yaml.v3"
)

var api string = "https://gaana.com/playlist/amreshwarsingh-ybddw-my2010ssongs"

// CookieYAML is the on-disk representation (YAML).
// Note: fields must be Exported for yaml.Unmarshal to populate them.
type CookieYAML struct {
	Name     string `yaml:"name"`
	Value    string `yaml:"value"`
	Domain   string `yaml:"domain,omitempty"`
	Path     string `yaml:"path,omitempty"`
	Expiry   int64  `yaml:"expiry,omitempty"` // unix seconds
	Secure   bool   `yaml:"secure,omitempty"`
	HttpOnly bool   `yaml:"httpOnly,omitempty"`
	SameSite string `yaml:"sameSite,omitempty"` // "Default"|"Lax"|"Strict"|"None"
}

type CookieFileYAML struct {
	Cookies []CookieYAML `yaml:"cookies"`
}

func cookiesFilePath() string {
	// keep your existing absolute location, but avoid repeating it everywhere
	return filepath.Join(
		`C:\Users\z004twvd\Documents\toSpotify\internal\apis`,
		"savecookie.yml",
	)
}

func CallAPI(client *http.Client) error {
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// headers/status
	log.Printf("status=%s content-type=%q", resp.Status, resp.Header.Get("Content-Type"))

	// payload (limit to 1MB so you don't spam the terminal)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	// save response locally
	file, err := os.Create("./response.html")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(bodyBytes)
	if err != nil {
		return err
	}
	saveCookies(client, resp)
	err = extarctNameofSongs(strings.NewReader(string(bodyBytes)), "./songs.txt")
	if err != nil {
		return err
	}

	return nil
}
func extarctNameofSongs(in io.Reader, outPath string) error {
	doc, err := goquery.NewDocumentFromReader(in)
	if err != nil {
		log.Fatalf("failed to create document from reader: %v", err)
	}

	// Scope to the playlist area inside mainContainer, then each song row.
	rows := doc.Find("div.mainContainer section.song-list ul._row.list_data")

	names := make([]string, 0, rows.Length()+1)
	names = append(names, "Song\tArtist(s)") // header
	rows.Each(func(_ int, row *goquery.Selection) {
		// Song title
		a := row.Find("div._tra a").First()
		if a.Length() == 0 {
			return
		}

		c := a.Clone()
		c.Find("span.new_premium, span.eicon").Remove()
		song := strings.TrimSpace(strings.Join(strings.Fields(c.Text()), " "))
		if song == "" {
			return
		}

		// Artist(s)
		artists := make([]string, 0, 4)
		row.Find("div._art a").Each(func(_ int, as *goquery.Selection) {
			name := strings.TrimSpace(strings.Join(strings.Fields(as.Text()), " "))
			if name != "" {
				artists = append(artists, name)
			}
		})
		artistStr := strings.Join(artists, ", ")

		// Write: song<TAB>artist(s)
		names = append(names, fmt.Sprintf("%s\t%s", song, artistStr))
	})

	out, err := os.Create(outPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	for _, n := range names {
		fmt.Fprintln(w, n)
	}
	if err := w.Flush(); err != nil {
		log.Fatalf("failed to flush output file: %v", err)
	}

	fmt.Printf("wrote %d songs to %s\n", len(names), outPath)
	return nil
}

// saveCookies writes cookies for rawURL from the client's jar into savecookie.yml as YAML.
func saveCookies(client *http.Client, resp *http.Response) error {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return nil
	}
	u := resp.Request.URL
	if u == nil {
		return nil
	}

	httpCookies := client.Jar.Cookies(u)
	out := CookieFileYAML{Cookies: make([]CookieYAML, 0, len(httpCookies))}
	for _, c := range httpCookies {
		out.Cookies = append(out.Cookies, CookieYAML{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expiry:   c.Expires.Unix(),
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: sameSiteToString(c.SameSite),
		})
	}

	b, err := yaml.Marshal(&out)
	if err != nil {
		return err
	}
	return os.WriteFile(cookiesFilePath(), b, 0644)
}

// SeedCookies reads savecookie.yml, converts to []*http.Cookie, and seeds the jar for rawURL.
func SeedCookies(client *http.Client, rawURL string) error {
	if client == nil || client.Jar == nil {
		return nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	b, err := os.ReadFile(cookiesFilePath())
	if err != nil {
		// no file yet = nothing to seed
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var doc CookieFileYAML
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return err
	}

	cookies := make([]*http.Cookie, 0, len(doc.Cookies))
	for _, yc := range doc.Cookies {
		c := &http.Cookie{
			Name:     yc.Name,
			Value:    yc.Value,
			Domain:   yc.Domain,
			Path:     yc.Path,
			Secure:   yc.Secure,
			HttpOnly: yc.HttpOnly,
			SameSite: stringToSameSite(yc.SameSite),
		}
		if yc.Expiry > 0 {
			c.Expires = time.Unix(yc.Expiry, 0)
		}
		cookies = append(cookies, c)
	}

	client.Jar.SetCookies(u, cookies)
	return nil
}

func sameSiteToString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return "Default"
	}
}

func stringToSameSite(s string) http.SameSite {
	switch s {
	case "Lax", "lax":
		return http.SameSiteLaxMode
	case "Strict", "strict":
		return http.SameSiteStrictMode
	case "None", "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}
