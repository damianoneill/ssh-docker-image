# ssh-docker-image

Command line application for pushing / installing a docker image on a remote ssh host.

## usage

```console
$ ssh-docker-image push -i bash:latest -d <user@host> -p xxxx
```

## help

```console
$ ssh-docker-image push -h
transfer and install image on remote host

Usage:
  ssh-docker-image push [flags]

Flags:
      --config string     config file (default is $HOME/ssh-docker-image.yaml)
  -d, --dest string       remote destination <user@host>
  -h, --help              help for push
  -i, --image string      image name and version <image:version>
  -l, --local string      local temporary directory (default "/tmp")
  -p, --password string   ssh password
  -r, --remote string     remote temporary directory (default "/tmp")
  -t, --timeout int       scp timeout in minutes (default 15)
```