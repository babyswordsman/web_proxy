package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"testing"
)

func TestUploadFile(t *testing.T) {
	buf := &bytes.Buffer{}
	dstWriter := multipart.NewWriter(buf)
	filePath := "/workspace/web_proxy/README.md"
	_, filename := path.Split(filePath)
	fileWriter, err := dstWriter.CreateFormFile("file", filename)
	if err != nil {
		fmt.Println("create form file err:", err.Error())
	}

	uploadFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println(err.Error())
	}
	len, err := io.Copy(fileWriter, uploadFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println("file len:", len)
	dstWriter.Close()
	
	req, err := http.NewRequest("POST", "http://120.26.95.192:8087/rag/upload", buf)
	if err != nil {
		fmt.Println(err.Error())
	}
	req.Header.Set("Content-Type", dstWriter.FormDataContentType())
	req.Header.Set("Cookie", "cookie_sess=MTcxNTk1OTQ3OHxEWDhFQVFMX2dBQUJFQUVRQUFEX21QLUFBQUVHYzNSeWFXNW5EQkFBRG1GcFgzTmxZWEpqYUY5MWMyVnlCbk4wY21sdVp3eHlBSEI3SWxWelpYSkpaQ0k2SWprek1tRmxaREV4TFRFME5qRXRNVEZsWmkwNU5HWmtMVEF3TVRZelpURTBZMkkzT1NJc0lsVnpaWEpPWVcxbElqb2lJaXdpVEdGemRIUnBiV1VpT2lJeU1ESTBMVEExTFRFM1ZESXpPakkwT2pNNExqVXlNakkzTkRVeU15c3dPRG93TUNKOXyCR-y69wH8gzS9AU9qz0vhxd3iACFKXGRZHdjWkgcJlg==")

	//rsp, err := http.Post("http://120.26.95.192:8007/rag/upload", dstWriter.FormDataContentType(), buf)
	if err != nil {
		fmt.Println("post err:", err.Error())
	}

	// step 2: use http.Transport to do request to real server.
	transport := http.DefaultTransport
	//不要打印，https的请求中有机密信息
	//log.Println(req.URL.String())

	rsp, err := transport.RoundTrip(req)
	if err != nil {
		fmt.Printf("error in roundtrip: %v", err)

	}

	content, err := io.ReadAll(rsp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(string(content))

}
