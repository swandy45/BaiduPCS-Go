package pcscommand

import (
	"fmt"
	"github.com/iikira/BaiduPCS-Go/pcsconfig"
	"github.com/iikira/BaiduPCS-Go/pcspath"
	"github.com/iikira/BaiduPCS-Go/pcsutil"
	fpath "path"
	"regexp"
	"strings"
)

var (
	// 通配符仅对 ? 和 * 起作用
	patternRE = regexp.MustCompile(`[\*\?]`)
)

// getAllAbsPaths 获取所有绝对路径
func getAllAbsPaths(paths ...string) (_paths []string, err error) {
	for k := range paths {
		p, err := parsePath(paths[k])
		if err != nil {
			return nil, err
		}
		_paths = append(_paths, p...)
	}
	return
}

// getAbsPath 使用通配符获取绝对路径, 返回值为第一个匹配结果, 获取错误将会返回 原路径 和 错误信息
func getAbsPath(path string) (string, error) {
	p, err := parsePath(path)
	if err != nil {
		return path, err
	}

	if len(p) >= 0 {
		return p[0], nil
	}
	return path, fmt.Errorf("未找到路径")
}

// parsePath 解析通配符
func parsePath(path string) (paths []string, err error) {
	pcsPath := pcspath.NewPCSPath(&pcsconfig.ActiveBaiduUser.Workdir, path)
	path = pcsPath.AbsPathNoMatch()

	if patternRE.MatchString(path) {
		// 递归
		paths = recurseParsePath(path)
		if len(paths) == 0 {
			return nil, fmt.Errorf("文件路径匹配失败, 请检查通配符")
		}

		return paths, nil
	}

	paths = []string{path}
	return
}

// recurseParsePath 递归解析通配符
func recurseParsePath(path string) (paths []string) {
	if !patternRE.MatchString(path) {
		// 检测路径是否存在
		_, err := info.FilesDirectoriesMeta(path)
		if err != nil {
			return nil
		}
		paths = []string{path}
		return
	}

	names := pcspath.SplitAll(path)
	namesLen := len(names)

	for k := range names {
		if !patternRE.MatchString(names[k]) {
			continue
		}

		pfiles, err := info.FilesDirectoriesList(strings.Join(names[:k], ""), false)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		// 多线程获取信息
		wg := pcsutil.NewWaitGroup(10)

		for k2 := range pfiles {
			wg.AddDelta()
			go func(k2 int) {
				ok, _ := fpath.Match(pcspath.EscapeBracketOne(names[k]), "/"+pfiles[k2].Filename)
				if ok {
					if k >= namesLen-1 {
						wg.Lock()
						paths = append(paths, pfiles[k2].Path) // 插入数据
						wg.Unlock()
					} else if pfiles[k2].Isdir {
						recPaths := recurseParsePath(pfiles[k2].Path + strings.Join(names[k+1:], ""))
						wg.Lock()
						paths = append(paths, recPaths...) // 插入数据
						wg.Unlock()
					}
				}

				wg.Done()
			}(k2)
		}

		wg.Wait()
		break
	}

	return
}
