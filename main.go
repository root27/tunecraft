package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"encoding/json"
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
					
					newId = id.split("v=")[1];
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

	fmt.Println("Server starting at port 9095")

	log.Fatal(http.ListenAndServe(":9095", nil))

}

func downloadAndExtractMp3(id string, output http.ResponseWriter) error {

	url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", id)

	callUrl := fmt.Sprintf("https://noembed.com/embed?url=%s", url)

	callRes, err := http.Get(callUrl)

	if err != nil {

		return err

	}

	resBody, err := io.ReadAll(callRes.Body)

	if err != nil {

		log.Printf("Error reading body: %+v", err)

		return err

	}

	defer callRes.Body.Close()

	var out map[string]interface{}

	_ = json.Unmarshal(resBody, &out)

	// Create pipes for communication
	ytdlRead, ytdlWrite := io.Pipe()
	ffmpegRead, ffmpegWrite := io.Pipe()

	// YouTube-DL Command
	ytdl := exec.Command("youtube-dl", url, "-o-")
	ytdl.Stdout = ytdlWrite
	ytdl.Stderr = os.Stderr

	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "-ab", "96000", "-vn", "-")
	ffmpeg.Stdin = ytdlRead
	ffmpeg.Stdout = ffmpegWrite
	ffmpeg.Stderr = os.Stderr

	output.Header().Set("Content-Type", "audio/mpeg")
	output.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.mp3", out["title"]))

	go func() {
		defer ytdlWrite.Close()
		ytdl.Run()

	}()

	go func() {
		defer ffmpegWrite.Close()
		ffmpeg.Run()
	}()

	_, err = io.Copy(output, ffmpegRead)
	if err != nil {
		return err
	}

	// Wait for FFmpeg to finish
	ffmpeg.Wait()

	return nil

}

func download(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")

	fmt.Println(id)

	err := downloadAndExtractMp3(id, w)

	if err != nil {

		http.Error(w, "Failed to download", http.StatusInternalServerError)
		return

	}

}
