package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// 設定：出力先ディレクトリ
const outputDir = "public/diary"
const dataDir = "data"

type DayEntry struct {
	Year    string
	Month   string
	Day     string
	Content template.HTML
	Path    string
}

func main() {
	// 1. データの収集
	entries, err := collectEntries(dataDir)
	if err != nil {
		panic(err)
	}

	// 2. 出力ディレクトリのクリーニングと作成
	os.RemoveAll("public")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}

	// 3. 階層構造データの作成
	tree := buildTree(entries)

	// 4. HTMLの生成

	// 4-1. 各日のページ
	for _, entry := range entries {
		html := renderTemplate("day", entry)
		saveFile(filepath.Join(outputDir, entry.Year, entry.Month, entry.Day+".html"), html)
	}

	// 4-2. 月のページ
	for year, months := range tree {
		for month, days := range months {
			data := map[string]interface{}{
				"Year": year, "Month": month, "Days": days,
			}
			html := renderTemplate("month", data)
			saveFile(filepath.Join(outputDir, year, month+".html"), html)
		}
	}

	// 4-3. 年のページ
	for year, months := range tree {
		// 月順ソート用
		var monthKeys []string
		for m := range months {
			monthKeys = append(monthKeys, m)
		}
		sort.Strings(monthKeys)

		data := map[string]interface{}{
			"Year": year, "Months": monthKeys,
		}
		html := renderTemplate("year", data)
		saveFile(filepath.Join(outputDir, year+".html"), html)
	}

	// 4-4. トップページ
	var years []string
	for y := range tree {
		years = append(years, y)
	}
	sort.Strings(years)

    var latestEntry *DayEntry
	if len(entries) > 0 {
		// 日付が新しい順にソート
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].Year != entries[j].Year {
				return entries[i].Year > entries[j].Year
			}
			if entries[i].Month != entries[j].Month {
				return entries[i].Month > entries[j].Month
			}
			return entries[i].Day > entries[j].Day
		})
		latestEntry = &entries[0]
	}

	data := map[string]interface{}{
        "Years": years,
        "Latest": latestEntry,
    }
	html := renderTemplate("index", data)
	saveFile(filepath.Join(outputDir, "index.html"), html)

	fmt.Println("生成完了: ./public ディレクトリを確認してください")
}

func collectEntries(root string) ([]DayEntry, error) {
	var entries []DayEntry

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),          // GitHub Flavored Markdown
		goldmark.WithRendererOptions(html.WithUnsafe()), // WithUnsafe: <img>タグなどをそのまま出力する
	)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			parts := strings.Split(filepath.ToSlash(path), "/")
			if len(parts) < 4 {
				return nil
			}
			year, month := parts[len(parts)-3], parts[len(parts)-2]
			day := strings.TrimSuffix(parts[len(parts)-1], ".md")

			content, _ := os.ReadFile(path)
			var buf bytes.Buffer
			
			// 作成したパーサー(md)を使って変換
			if err := md.Convert(content, &buf); err != nil {
				return err
			}

			entries = append(entries, DayEntry{
				Year: year, Month: month, Day: day,
				Content: template.HTML(buf.String()),
			})
		}
		return nil
	})
	return entries, err
}

func buildTree(entries []DayEntry) map[string]map[string][]DayEntry {
	tree := make(map[string]map[string][]DayEntry)
	for _, e := range entries {
		if tree[e.Year] == nil {
			tree[e.Year] = make(map[string][]DayEntry)
		}
		tree[e.Year][e.Month] = append(tree[e.Year][e.Month], e)
	}
	for y := range tree {
		for m := range tree[y] {
			sort.Slice(tree[y][m], func(i, j int) bool {
				return tree[y][m][i].Day < tree[y][m][j].Day
			})
		}
	}
	return tree
}

func saveFile(path string, content string) {
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)
	os.WriteFile(path, []byte(content), 0644)
}

func renderTemplate(kind string, data interface{}) string {
	const tplString = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>日記</title>
	<style>
		body { font-family: sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; line-height: 1.6; }
		img { max-width: 100%; height: auto; } /* 画像がはみ出さないように */
		a { color: #0066cc; }
		blockquote { border-left: 4px solid #ccc; margin: 0; padding-left: 16px; color: #555; }
	</style>
</head>
<body>
	{{if eq .Kind "index"}}
		<h2>日記</h2>
        {{if .Data.Latest}}
            <h3>最新</h3>
			<a href="{{.Data.Latest.Year}}/{{.Data.Latest.Month}}/{{.Data.Latest.Day}}.html">
				{{.Data.Latest.Year}}年{{.Data.Latest.Month}}月{{.Data.Latest.Day}}日
			</a>
		{{end}}

		<h3>アーカイブ</h3>
		<ul>{{range .Data.Years}}<li><a href="{{.}}.html">{{.}}年</a></li>{{end}}</ul>
	{{else if eq .Kind "year"}}
        <nav>
            <a href="/diary/index.html">トップ</a>
        </nav>
        <hr>
		<h2>{{.Data.Year}}</h2>
		<ul>{{range .Data.Months}}<li><a href="{{$.Data.Year}}/{{.}}.html">{{.}}月</a></li>{{end}}</ul>
	{{else if eq .Kind "month"}}
        <nav>
            <a href="/diary/index.html">トップ</a>
            /
            <a href="/diary/{{.Data.Year}}.html">{{.Data.Year}}</a>
        </nav>
        <hr>
		<h2>{{.Data.Year}}/{{.Data.Month}}</h2>
		<ul>{{range .Data.Days}}<li><a href="{{.Month}}/{{.Day}}.html">{{.Day}}日</a></li>{{end}}</ul>
	{{else if eq .Kind "day"}}
        <nav>
            <a href="/diary/index.html">トップ</a>
            /
            <a href="/diary/{{.Data.Year}}.html">{{.Data.Year}}</a>
            /
            <a href="/diary/{{.Data.Year}}/{{.Data.Month}}.html">{{.Data.Month}}</a>
        </nav>
        <hr>
		<h2>{{.Data.Year}}/{{.Data.Month}}/{{.Data.Day}}</h2>
		<div>{{.Data.Content}}</div>
	{{end}}
</body>
</html>`
	
	t, _ := template.New("base").Parse(tplString)
	var buf bytes.Buffer
	wrapper := struct {
		Kind string
		Data interface{}
	}{Kind: kind, Data: data}
	
	t.Execute(&buf, wrapper)
	return buf.String()
}
