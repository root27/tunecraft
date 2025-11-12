package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func homePage(w http.ResponseWriter, r *http.Request) {
	html := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>TuneCraft - Youtube2Mp3</title>
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<style>
			body {
				font-family: Arial, sans-serif;
				margin:0;
				padding:0;
			}
			h1 {
				text-align: center;
				margin-bottom: 20px;
			}
			.container {
				width:90%;
				max-width: 700px;
				margin: 100px auto;
				padding: 20px;
				background-color: #fff;
				box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
				border-radius: 8px;
				text-align: center;
			}
			form {
				display: flex;
				flex-direction: column;
				align-items: center;
				gap: 10px;
			}
			form label {
				font-size: 1.2rem;
				color: #555;
				text-align: left;
			}
			form input[type="text"] {
				width:35%;
				max-width: 500px;
				margin: 0 auto;
				border: 1px solid #ccc;
				border-radius: 4px;
				padding: 10px;
				font-size: 1em;
			}
			form input[type="submit"] {
				width:30%;
				max-width: 500px;
				padding: 10px;
				font-size: 1.1rem;
				background-color: #4CAF50;
				color: white;
				border: none;
				border-radius: 4px;
				cursor: pointer;
			}
			form input[type="submit"]:hover {
				background-color: #45a049;
			}
		</style>
		<body>
			<h1>TuneCraft: YouTube to MP3 Converter</h1>
			<div class="container">
				<form id="formaction">
					<label for="url">Enter YouTube Video ID or Url:</label>
					<input type="text" id="url" name="url" required>
					<input type="submit" value="Convert to MP3">
				</form>
			</div>
		
		<script>
			document.getElementById("formaction").addEventListener("submit", function(e) {
				e.preventDefault();
				const id = document.getElementById("url").value;
					
				if (id.includes("youtube.com")) {
					let newId = id.split("v=")[1];
					if (newId && newId.includes("&")) {
						newId = newId.split("&")[0];
					}
					window.location.href = "/download?id=" + newId;
				} else {
					window.location.href = "/download?id=" + id;
				}
			});
		</script>
		</body>
		</html>`
	fmt.Fprint(w, html)
}

func main() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/download", download)

	port := os.Getenv("PORT")
	if port == "" {
		port = "9095"
	}

	fmt.Printf("Server starting at port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func downloadAndExtractMp3(id string, output http.ResponseWriter) error {
	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)
	infoURL := fmt.Sprintf("https://noembed.com/embed?url=%s", url)

	resp, err := http.Get(infoURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var meta map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return err
	}

	title := "audio"
	if t, ok := meta["title"].(string); ok {
		title = strings.NewReplacer("/", "-", "\\", "-", "\"", "").Replace(t)
	}

	output.Header().Set("Content-Type", "audio/mpeg")
	output.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.mp3\"", title))

	apiURL := fmt.Sprintf("https://ytdlp.online/stream?command=%%20-x%%20--audio-format%%20mp3%%20%s", strings.ReplaceAll(url, ":", "%3A"))

	sseResp, err := http.Get(apiURL)
	if err != nil {
		return err
	}
	defer sseResp.Body.Close()

	scanner := bufio.NewScanner(sseResp.Body)
	downloadLink := ""
	re := regexp.MustCompile(`href="([^"]+\.mp3)"`)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			log.Println(data)

			if matches := re.FindStringSubmatch(data); len(matches) > 1 {
				downloadLink = "https://ytdlp.online" + matches[1]
				log.Println("Found download link:", downloadLink)
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if downloadLink == "" {
		return fmt.Errorf("failed to get MP3 download link")
	}

	mp3Resp, err := http.Get(downloadLink)
	if err != nil {
		return err
	}
	defer mp3Resp.Body.Close()

	_, err = io.Copy(output, mp3Resp.Body)
	if err != nil {
		return err
	}

	log.Printf("Streaming completed for: %s", title)
	return nil
}

func download(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing video ID", http.StatusBadRequest)
		return
	}

	log.Printf("Processing download request for ID: %s", id)

	err := downloadAndExtractMp3(id, w)
	if err != nil {
		log.Printf("Download failed: %v", err)
		http.Error(w, "Failed to download", http.StatusInternalServerError)
		return
	}
}
