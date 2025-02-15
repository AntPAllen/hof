package fmt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"

	"github.com/hofstadter-io/hof/cmd/hof/flags"
	"github.com/hofstadter-io/hof/lib/docker"
	"github.com/hofstadter-io/hof/lib/repos/cache"
	"github.com/hofstadter-io/hof/lib/yagu"
)

var dataFileExtns = map[string]struct{}{
	".cue": struct{}{},
	".yml": struct{}{},
	".yaml": struct{}{},
	".json": struct{}{},
	".toml": struct{}{},
	".xml": struct{}{},
}

type formatGroup struct {
	// original arg from cli
	orig string

	// filepath from cli arg
	path string

	// formatter from cli arg, if present
	formatter string

	// glob passed to find files
	glob string

	// files found
	files []string
}

func Run(args []string, rflags flags.RootPflagpole, cflags flags.FmtFlagpole) (err error) {
	// we need to build up a struct to iterate over
	gs := []formatGroup{}

	// cleanup args
	for _, arg := range args {
		if rflags.Verbosity > 2 {
			fmt.Println(args)
		}

		g := formatGroup {
			orig: arg,
		}

		// extract formatter settings
		parts := strings.Split(arg, "@")
		g.path = parts[0]
		if len(parts) > 2 {
			return fmt.Errorf("bad arg %q", arg)
		}
		if len(parts) == 2 {
			g.formatter = parts[1]
		}

		// default for single files
		g.glob = g.path
		// if path is a dir and has no globs already, make it globby for recursion (simplify UX for a common case)
		if !strings.Contains(g.path, "*") {
			info, err := os.Stat(g.path)
			if err != nil {
				return err
			}

			// if the arg is a dir, assume recursive and adjust the glob to do so
			if info.IsDir() {
				// fully traverse directories
				glob := "**/*"
				// slash fix
				if arg[len(arg)-1] != '/' {
					glob = "/" + glob
				}
				g.glob = g.path + glob
			}
		}

		if rflags.Verbosity > 3 {
			fmt.Println(g)
		}

		// find files from glob
		if strings.Contains(g.glob, "*") {
			g.files, err = yagu.FilesFromGlobs([]string{g.glob})
			if err != nil {
				return err
			}
		} else {
			g.files = []string{g.glob}
		}

		if rflags.Verbosity > 2 {
			fmt.Println(g)
		}

		gs = append(gs, g)
	}

	errCount := 0
	// loop over groups
	for _, g := range gs {
		// filter files (data & dirs)
		files := []string{}
		for _, file := range g.files {
			info, err := os.Stat(file)
			if err != nil {
				return err
			}
			if info.IsDir() {
				continue
			}	
			if !cflags.Data {
				ext := filepath.Ext(file)
				if _, ok := dataFileExtns[ext]; ok {
					continue
				}
			}
			files = append(files,file)
		}

		// if verbosity great enough?
		fmt.Printf("formatting %d file(s) from %s\n", len(files), g.orig)

		for _, file := range files {
			if rflags.Verbosity > 0 {
				fmt.Println(file)
			}

			// duplicated, but we need the info to preserve mode below
			info, err := os.Stat(file)
			if err != nil {
				return err
			}

			content, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			
			// todo, add flags for fmtr & config
			fmtd, err := FormatSource(file, content, g.formatter, nil, cflags.Data)
			if err != nil {
				fmt.Println("while formatting source:", err)
				errCount += 1
				continue
			}

			err = os.WriteFile(file, fmtd, info.Mode())
			if err != nil {
				return err
			}
		}
	}

	if errCount > 0 {
		return fmt.Errorf("encountered %v errors while formatting", errCount)
	}

	return nil
}

func Start(fmtr string, replace bool) error {
	err := updateFormatterStatus()
	if err != nil {
		return err
	}

	// override the default version
	ver := defaultVersion
	parts := strings.Split(fmtr, "@")
	if len(parts) == 2 {
		fmtr, ver = parts[0], parts[1]
	}

	if fmtr == "" {
		fmtr = "all"
	}

	if ver == "latest" || ver == "next" {
		v, err := cache.GetLatestTag("github.com/hofstadter-io/hof", ver == "next")
		if err != nil {
			return err
		}
		ver = v
	}

	startFmtr := func(name, ver string) error {
		fmt.Println("starting:", name, ver)
		fmtr := formatters[name]
		// what other statuses do we need to check here? (maybe none)
		if fmtr.Status == "exited" {
			err := docker.StopContainer(fmt.Sprintf("hof-fmt-%s", name))
			if err != nil {
				return err
			}
		}
		return docker.StartContainer(
			fmt.Sprintf("%s/fmt-%s:%s", CONTAINER_REPO, name, ver),
			fmt.Sprintf("hof-fmt-%s", name),
			fmtrEnvs[name],
			replace,
		)
	}

	waitFmtr := func(name string) error {
		fmtr := formatters[name]

		// wait for running & ready
		err = fmtr.WaitForRunning(10, time.Second)
		if err != nil {
			return err
		}
		err = fmtr.WaitForReady(30, time.Second)
		if err != nil {
			return err
		}

		return nil
	}

	if fmtr == "all" {
		for _, name := range fmtrNames {
			err = startFmtr(name, ver)
			if err != nil {
				fmt.Println(err)
			}
		}
		for _, name := range fmtrNames {
			err = waitFmtr(name)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		err = startFmtr(fmtr, ver)
		if err != nil {
			return err
		}
		err = waitFmtr(fmtr)
		if err != nil {
			return err
		}
	}

	// TODO, add alive command and wait for ready
	// time.Sleep(2000*time.Millisecond)

	return nil
}

func Stop(fmtr string) error {
	if fmtr == "" {
		fmtr = "all"
	}

	if fmtr == "all" {
		for _, name := range fmtrNames {
			err := docker.StopContainer(fmt.Sprintf("hof-fmt-%s", name))
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		return docker.StopContainer(fmt.Sprintf("hof-fmt-%s", fmtr))
	}
	return nil
}

func Pull(fmtr string) error {
	// override the default version
	ver := defaultVersion
	parts := strings.Split(fmtr, "@")
	if len(parts) == 2 {
		fmtr, ver = parts[0], parts[1]
	}
	if ver == "latest" || ver == "next" {
		v, err := cache.GetLatestTag("github.com/hofstadter-io/hof", ver == "next")
		if err != nil {
			return err
		}
		ver = v
	}

	if ver == "dirty" {
		return fmt.Errorf("%s: You have local changes to hof, run 'make formatters' instead", fmtr)
	}

	if fmtr == "" {
		fmtr = "all"
	}

	if fmtr == "all" {
		for _, name := range fmtrNames {
			ref := fmt.Sprintf("%s/fmt-%s:%s", CONTAINER_REPO, name, ver)
			err := docker.PullImage(ref)
			if err != nil {
				fmt.Println(err)
			}
		}
	} else {
		ref := fmt.Sprintf("%s/fmt-%s:%s", CONTAINER_REPO, fmtr, ver)
		return docker.PullImage(ref)
	}
	return nil
}

func Info(which string) (err error) {
	err = updateFormatterStatus()
	if err != nil {
		return err
	}

	return printAsTable(
		[]string{"Name", "Status", "Port", "Image", "Available"},
		func(table *tablewriter.Table) ([][]string, error) {
			var rows = make([][]string, 0, len(fmtrNames))
			// fill with data
			for _,f := range fmtrNames {
				fmtr := formatters[f]

				if which != "" {
					if !strings.HasPrefix(fmtr.Name, which) {
						continue
					}
				}

				
				if fmtr.Container != nil {
					rows = append(rows, []string{
						fmtr.Name,
						fmtr.Container.Status,
						fmtr.Port,
						fmtr.Container.Image,
						fmt.Sprint(fmtr.Available),
					})
				} else {
					img := ""
					if len(fmtr.Images) > 0 {
						if len(fmtr.Images[0].RepoTags) > 0 {
							img = fmtr.Images[0].RepoTags[0]
						}
					}
					rows = append(rows, []string{
						fmtr.Name,
						"", "", img,
						fmt.Sprint(fmtr.Available),
					})
				}
			}
			return rows, nil
		},
	)

	return nil
}

func updateFormatterStatus() error {

	images, err := docker.GetImages(fmt.Sprintf("%s/fmt-", CONTAINER_REPO))
	if err != nil {
		return err
	}
	containers, err := docker.GetContainers("hof-fmt-")
	if err != nil {
		return err
	}

	// reset formatters
	for _, fmtr := range formatters {
		fmtr.Running = false
		fmtr.Container = nil
		fmtr.Available = make([]string, 0)
	}

	for _, image := range images {
		added := false
		for _, tag := range image.RepoTags {
			parts := strings.Split(tag, ":")
			repo, ver := parts[0], parts[1]
			name := strings.TrimPrefix(repo, fmt.Sprintf("%s/fmt-", CONTAINER_REPO))
			fmtr := formatters[name]
			fmtr.Available = append(fmtr.Available, ver)
			if !added {
				fmtr.Images = append(fmtr.Images, &image)
				added = true
			}
		}
	}


	for _, container := range containers {
		// extract name
		name := container.Names[0]
		name = strings.TrimPrefix(name, "/" + ContainerPrefix)

		// get fmtr
		fmtr := formatters[name]

		fmtr.Status = container.State

		// determine the container status
		if container.State == "running" {
			fmtr.Running = true
		} else {
			fmtr.Running = false
		}

		p := 100000
		for _, port := range container.Ports {
			P := int(port.PublicPort)
			if P < p {
				p = P
			}
		}

		if p != 100000 {
			fmtr.Port = fmt.Sprint(p)
		}

		// save container to fmtr
		c := container
		fmtr.Container = &c

		formatters[name] = fmtr
	}

	return nil
}

func defaultTableFormat(table *tablewriter.Table) {
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("  ") // pad with tabs
	table.SetNoWhiteSpace(true)
}

type dataPrinter func(table *tablewriter.Table) ([][]string, error)

func printAsTable(headers []string, printer dataPrinter) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)

	defaultTableFormat(table)

	rows, err := printer(table)
	if err != nil {
		return err
	}

	table.AppendBulk(rows)

	// render
	table.Render()

	return nil
}

