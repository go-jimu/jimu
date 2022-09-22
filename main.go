package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	TemplateURL    = "https://codeload.github.com/go-jimu/template/zip/refs/heads/master"
	UnzipDirectory = "template-master"
)

var (
	contentToReplace = []byte("github.com/go-jimu/template")
)

type (
	// Project 项目信息
	Project struct {
		BinFile string // 编译的二进制文件名称，比如 app
		Module  string // 模块名称，比如 github.com/go-jimu/web
	}
)

func DownloadTemplate(template string) (string, error) {
	resp, err := http.Get(template)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fn := strconv.Itoa(int(time.Now().Unix())) + ".zip"
	fn = filepath.Join(os.TempDir(), fn)

	f, err := os.Create(fn)
	if err != nil {
		return "", err
	}

	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}

	return fn, nil
}

func Unzip(file string) error {
	reader, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, f := range reader.File {
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(f.Name, os.ModePerm)
			continue
		}

		f1, err := f.Open()
		if err != nil {
			return err
		}
		err = os.MkdirAll(filepath.Dir(f.Name), os.ModePerm)
		if err != nil {
			return err
		}
		f2, err := os.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(f2, f1)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProjectSetting() *Project {
	project := new(Project)
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("输入项目Module（如 github.com/go-jimu/project）：")
	text, _ := reader.ReadString('\n')
	project.Module = strings.Trim(text, "\n")
	if project.Module == "" {
		panic("module不能为空")
	}

	fmt.Print("输入项目编译二进制文件名称(如 app):")
	text, _ = reader.ReadString('\n')
	project.BinFile = strings.Trim(text, "\n")
	if project.BinFile == "" {
		panic("二进制文件名不能为空")
	}

	fmt.Printf("请确认配置（按Y确认）：%v", project)
	text, _ = reader.ReadString('\n')
	if !strings.HasPrefix(strings.ToLower(text), "y") {
		os.Exit(1)
	}
	return project
}

func RenderTemplateProject(dst string, proj *Project) error {
	return filepath.Walk(dst, func(path string, info os.FileInfo, _ error) error {
		log.Println(dst, path)
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		data = bytes.Replace(data, contentToReplace, []byte(proj.Module), -1)
		if info.Name() == "Dockerfile" {
			data = bytes.Replace(data, []byte("template"), []byte(proj.BinFile), -1)
		}

		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, info.Mode())
		if err != nil {
			panic(err)
		}
		defer file.Close()
		if _, err = file.Write(data); err != nil {
			panic(err)
		}
		return nil
	})
}

func main() {
	proj := ProjectSetting()
	log.Println("开始下载模板文件")
	f, err := DownloadTemplate(TemplateURL)
	if err != nil {
		panic(err)
	}
	log.Println(f)

	log.Println("解压缩模板文件")
	err = Unzip(f)
	if err != nil {
		panic(err)
	}

	log.Println("开始渲染模板")
	err = RenderTemplateProject(UnzipDirectory, proj)
	if err != nil {
		panic(err)
	}

	ds := strings.Split(proj.Module, "/")
	log.Println("重命名项目" + ds[len(ds)-1])
	if err = os.Rename(UnzipDirectory, ds[len(ds)-1]); err != nil {
		panic(err)
	}

	log.Println("完成，小的退下了")
}

