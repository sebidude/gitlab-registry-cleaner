```
usage: gitlab-registry-cleaner [<flags>] <command> [<args> ...]

Flags:
      --help         Show context-sensitive help (also try --help-long and --help-man).
  -t, --token=TOKEN  Gitlab access token

Commands:
  help [<command>...]
    Show help.

  show repos <project>
    Show repos of project

  show tags <project> [<repository>]
    Show tags in repository

  show runners
    Show offline group-runners

  clean repo [<flags>] <project> [<repository>]
    Cleanup tags in a repository

  clean all [<flags>] <account>
    Cleanup tags in all projects of a user/group

  clean runners
    Delete offline group-runners
```
