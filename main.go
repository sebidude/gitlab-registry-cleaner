package main

import (
	"errors"
	"log"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	gitlabtoken string
	repository  string
	project     string
	nameregex   string
	account     string

	keep = -1
)

type Client struct {
	Client *gitlab.Client
}

func main() {

	kingpin.Flag("token", "Gitlab access token").Short('t').Envar("GITLAB_TOKEN").StringVar(&gitlabtoken)

	show := kingpin.Command("show", "Show objects")
	showRepo := show.Command("repos", "Show repos of project")
	showRepo.Arg("project", "Project Name (user/project or group/project)").Required().StringVar(&project)
	showTags := show.Command("tags", "Show tags in repository")
	showTags.Arg("project", "Project Name (user/project or group/project)").Required().StringVar(&project)
	showTags.Arg("repository", "Name of the repository").Default("").StringVar(&repository)
	showRunners := show.Command("runners", "Show offline group-runners")

	clean := kingpin.Command("clean", "Cleanup objects")
	cleanRepo := clean.Command("repo", "Cleanup tags in a repository")
	cleanRepo.Arg("project", "Project Name (user/project or group/project)").Required().StringVar(&project)
	cleanRepo.Arg("repository", "Name of the repository").Default("").StringVar(&repository)
	cleanRepo.Flag("keep", "Keep the latest N tags").Short('k').IntVar(&keep)
	cleanRepo.Flag("nameregex", "Regex of the tag names to be cleaned up.").Default(".*").Short('n').StringVar(&nameregex)
	cleanAllRepos := clean.Command("all", "Cleanup tags in all projects of a user/group")
	cleanAllRepos.Arg("account", "Name of user or group").Required().StringVar(&account)
	cleanAllRepos.Flag("keep", "Keep the latest N tags").Short('k').IntVar(&keep)
	cleanAllRepos.Flag("nameregex", "Regex of the tag names to be cleaned up.").Default(".*").Short('n').StringVar(&nameregex)
	cleanRunners := clean.Command("runners", "Delete offline group-runners")

	auto := kingpin.Command("auto", "Automatable mode (clean all + clean runners)")
	auto.Arg("account", "Name of user or group").Required().StringVar(&account)
	auto.Flag("keep", "Keep the latest N tags").Short('k').IntVar(&keep)
	auto.Flag("nameregex", "Regex of the tag names to be cleaned up.").Default(".*").Short('n').StringVar(&nameregex)

	operation := kingpin.Parse()

	c := new(Client)
	c.Client = gitlab.NewClient(nil, gitlabtoken)

	switch operation {
	case showRepo.FullCommand():
		repos, err := c.GetRegistriesByProject(project)
		if err != nil {
			log.Fatal(err)
		}
		for _, r := range repos {
			log.Printf("%s %s\n", project, r.Name)
		}

	case showTags.FullCommand():
		tags, err := c.GetRegistriesTagsByProject(project, repository)
		if err != nil {
			log.Fatal(err)
		}
		for _, t := range tags {
			log.Printf("%s\n", t.Location)
		}

	case showRunners.FullCommand():
		runners, err := c.GetRunners(account)
		if err != nil {
			log.Fatal(err)
		}
		for _, r := range runners {
			log.Printf("runner id %d is %s\n", r.ID, r.Status)
		}

	case cleanRepo.FullCommand():
		err := c.CleanUpRepositoryTags(project, repository)
		if err != nil {
			log.Fatal(err)
		}

	case cleanAllRepos.FullCommand():
		err := c.CleanUpAllProjectRegistries(account)
		if err != nil {
			log.Fatal(err)
		}

	case cleanRunners.FullCommand():
		err := c.CleanUpRunners()
		if err != nil {
			log.Fatal(err)
		}

	case auto.FullCommand():
		err := c.CleanUpAllProjectRegistries(account)
		if err != nil {
			log.Fatal(err)
		}
		err = c.CleanUpRunners()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (c *Client) CleanUpRepositoryTags(project, name string) error {

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
		return err
	}

	for _, r := range repos {
		if r.Name == name {
			resp, err := c.Client.ContainerRegistry.DeleteRegistryRepositoryTags(project, r.ID, &opt)
			if err != nil {
				log.Println(err)
			}

			log.Println(resp.StatusCode, project, r.Name)
			return nil
		}
	}

	return errors.New("nothing found to delete.")
}

func (c *Client) CleanUpAllProjectRegistries(account string) error {
	opt := gitlab.ListGroupProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	for {
		// Get the first page with projects.
		projects, resp, err := c.Client.Groups.ListGroupProjects(account, &opt)
		if err != nil {
			return err
		}

		for _, p := range projects {
			repos, err := c.GetRegistriesByProject(p.PathWithNamespace)
			if err != nil {
				return err
			}

			for _, r := range repos {
				parts := strings.Split(r.Path, "/")
				subrepo := ""

				if len(parts) == 3 {
					subrepo = parts[2]
				}

				err = c.CleanUpRepositoryTags(p.PathWithNamespace, subrepo)

				if err != nil {
					return err
				}
			}
		}

		// Exit the loop when we've seen all pages.
		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		// Update the page number to get the next page.
		opt.Page = resp.NextPage

	}

	return nil
}

func (c *Client) GetRegistriesTagsByProject(project, name string) ([]*gitlab.RegistryRepositoryTag, error) {

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

	return nil, errors.New("nothing to list - maybe you need to specify the repository name? Check with 'show repos'.")
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

func (c *Client) GetRunners(account string) ([]*gitlab.Runner, error) {
	opt := gitlab.ListRunnersOptions{}
	runners, _, err := c.Client.Runners.ListRunners(&opt)
	if err != nil {
		return nil, err
	}

	if len(runners) == 0 {
		log.Println("no runners found")
		return nil, nil
	}

	return runners, nil
}

func (c *Client) CleanUpRunners() error {
	statusfilter := "offline"
	opt := gitlab.ListRunnersOptions{
		Scope: &statusfilter,
	}

	runners, _, err := c.Client.Runners.ListRunners(&opt)
	if err != nil {
		return err
	}

	if len(runners) == 0 {
		log.Println("no offline runners found")
		return nil
	}

	for _, runner := range runners {
		delResponse, err := c.Client.Runners.RemoveRunner(runner.ID)

		if err != nil {
			log.Printf("%d deleting runner with id %d failed: %s\n", delResponse.StatusCode, runner.ID, err)
		} else {
			log.Printf("%d runner with id %d deleted\n", delResponse.StatusCode, runner.ID)
		}
	}

	return nil
}
