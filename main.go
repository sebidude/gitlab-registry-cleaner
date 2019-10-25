package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kingpin"
	gitlab "github.com/xanzy/go-gitlab"
)

var (
	gitlabtoken string
	objtype     string
	repository  string
	project     string
	nameregex   string

	keep = -1
)

type Client struct {
	Client *gitlab.Client
}

func main() {

	app := kingpin.New(os.Args[0], "gitlab registry cleaner")
	app.Flag("token", "Gitlab access token").Short('t').Envar("GITLAB_TOKEN").StringVar(&gitlabtoken)

	show := app.Command("show", "repos of group.")
	show.Arg("object", "What to show").Default("repo").StringVar(&objtype)
	show.Arg("project", "Project Name (user/project or group/project)").Default("-").StringVar(&project)
	show.Arg("repository", "Name of the repository").StringVar(&repository)

	clean := app.Command("clean", "Cleanup tags from repository.")
	clean.Arg("project", "Project Name (user/project or group/project)").Default("-").StringVar(&project)
	clean.Arg("repository", "The repository to operate on.").Default("").StringVar(&repository)
	clean.Flag("keep", "Keep the latest N tags").Short('k').IntVar(&keep)
	clean.Flag("nameregex", "Regex of the tag names to be cleaned up.").Default(".*").Short('n').StringVar(&nameregex)

	operation := kingpin.MustParse(app.Parse(os.Args[1:]))

	c := new(Client)
	c.Client = gitlab.NewClient(nil, gitlabtoken)

	switch operation {
	case "show":

		if strings.HasPrefix(objtype, "repo") {
			repos, err := c.GetRegistriesByProject(project)
			if err != nil {
				fmt.Println(err)
			}
			for _, r := range repos {
				fmt.Printf("%s %s\n", project, r.Name)
			}
		}

		if objtype == "tags" {
			tags, err := c.GetRegistriesTagsByProject(project, repository)
			if err != nil {
				fmt.Println(err)
			}
			for _, t := range tags {
				fmt.Printf("%s\n", t.Location)
			}

		}

	case "clean":
		resp, err := c.CleanUpRepositoryTags(project, repository)
		if err != nil {
			fmt.Println(err)
			break
		}
		fmt.Println(resp.StatusCode)

	}

}

func (c *Client) CleanUpRepositoryTags(project, name string) (*gitlab.Response, error) {

	opt := gitlab.DeleteRegistryRepositoryTagsOptions{}
	opt.NameRegexp = &nameregex

	if keep != -1 {
		opt.KeepN = &keep
	}

	if nameregex == ".*" && keep == -1 {
		keep = 5
		opt.KeepN = &keep
	}

	repos, err := c.GetRegistriesByProject(project)
	if err != nil {
		return nil, err
	}

	for _, r := range repos {
		fmt.Printf("Checking matching name: %s <==> %s\n", r.Name, name)
		fmt.Printf("repo: %#v\n", r.Name)
		if r.Name == name {
			resp, err := c.Client.ContainerRegistry.DeleteRegistryRepositoryTags(project, r.ID, &opt)
			if err != nil {
				return nil, err
			}
			return resp, nil
		}
	}
	return nil, fmt.Errorf("nothing found to delete.")
}

func (c *Client) GetRegistriesTagsByProject(project, name string) ([]*gitlab.RegistryRepositoryTag, error) {
	//denic-id/c2id-auth-endpoint/dns-lookup-ms
	repos, err := c.GetRegistriesByProject(project)
	if err != nil {
		return nil, err
	}
	for _, r := range repos {
		if r.Name == name {
			opt := gitlab.ListRegistryRepositoryTagsOptions{
				PerPage: 100,
				Page:    1,
			}
			var rtags []*gitlab.RegistryRepositoryTag
			var err error
			for {
				// Get the first page with projects.
				tags, resp, err := c.Client.ContainerRegistry.ListRegistryRepositoryTags(project, r.ID, &opt)
				if err != nil {
					return nil, err
				}

				// List all the projects we've found so far.
				rtags = append(rtags, tags...)

				// Exit the loop when we've seen all pages.
				if resp.CurrentPage >= resp.TotalPages {
					break
				}

				// Update the page number to get the next page.
				opt.Page = resp.NextPage
			}
			return rtags, err

		}
	}

	return nil, fmt.Errorf("nothing to list")
}

func (c *Client) GetRegistriesByProject(name string) ([]*gitlab.RegistryRepository, error) {
	opt := gitlab.ListRegistryRepositoriesOptions{
		PerPage: 100,
		Page:    1,
	}
	var rrepos []*gitlab.RegistryRepository
	var err error
	for {
		// Get the first page with projects.
		repos, resp, err := c.Client.ContainerRegistry.ListRegistryRepositories(name, &opt)
		if err != nil {
			return nil, err
		}

		// List all the projects we've found so far.
		rrepos = append(rrepos, repos...)

		// Exit the loop when we've seen all pages.
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// Update the page number to get the next page.
		opt.Page = resp.NextPage
	}
	return rrepos, err

}
