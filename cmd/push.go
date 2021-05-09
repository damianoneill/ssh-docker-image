package cmd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"

	"github.com/docker/docker/client"
)

// pushCmd represents the push command
var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "transfer and install image on remote host",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// eng-registry.juniper.net/packet-optical/trust/network
		filename := strings.Replace(strings.Replace(viper.GetString("image"), ":", "_", 1), "/", "_", -1) + ".tar.gz"
		fmt.Println(">>> saving image locally: " + viper.GetString("local") + "/" + filename)
		err = saveImage(viper.GetString("image"), viper.GetString("local")+"/"+filename)
		if err != nil {
			return
		}
		sshConfig := &ssh.ClientConfig{
			User:            strings.Split(viper.GetString("dest"), "@")[0],
			Auth:            []ssh.AuthMethod{ssh.Password(viper.GetString("password"))},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // nolint:gosec
		}
		// username@host
		host := strings.Split(viper.GetString("dest"), "@")[1]
		connection, err := ssh.Dial("tcp", host+":22", sshConfig)
		if err != nil {
			return fmt.Errorf("failed to dial: %s", err)
		}
		defer connection.Close()

		fmt.Println(">>> scping local image to : " + viper.GetString("dest") + ":" + viper.GetString("remote") + "/" + filename + ", be patient")
		err = scpFile(viper.GetString("local")+"/"+filename, viper.GetString("remote")+"/"+filename, connection)
		if err != nil {
			return err
		}
		fmt.Println(">>> running docker load command for image: " +
			viper.GetString("remote") + "/" +
			filename + " on host: " + viper.GetString("dest"))
		return runCommand("docker load < "+viper.GetString("remote")+"/"+filename, connection)
	},
}

func saveImage(image, filename string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create docker api client: %s", err)
	}

	// save the image
	out, err := cli.ImageSave(ctx, []string{image})
	if err != nil {
		return fmt.Errorf("failed to save docker image: %s", err)
	}
	defer out.Close()

	// open file for writing
	in, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %s", err)
	}
	defer in.Close()

	// create gzip writer
	archiver := gzip.NewWriter(in)
	archiver.Name = filename
	defer archiver.Close()

	// write local tar.gz
	_, err = io.Copy(archiver, out)

	return err
}

func scpFile(local, remote string, connection *ssh.Client) error {
	c, err := scp.NewClientBySSH(connection)
	if err != nil {
		return fmt.Errorf("error creating new SSH session from existing connection: %s", err)
	}
	defer c.Close()
	// set the timeout to 60 minutes
	c.Timeout = time.Duration(viper.GetInt("timeout")) * time.Minute

	f, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("problem opening local file: %s", err)
	}
	defer f.Close()

	return c.CopyFile(f, remote, "0655")
}

func runCommand(cmd string, conn *ssh.Client) (err error) {
	sess, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create new session: %s", err)
	}
	defer sess.Close()
	sessStdOut, err := sess.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get std out pipe: %s", err)
	}
	go io.Copy(os.Stdout, sessStdOut) // nolint:errcheck
	sessStderr, err := sess.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get std err pipe: %s", err)
	}
	go io.Copy(os.Stderr, sessStderr) // nolint:errcheck
	err = sess.Run(cmd)               // eg., /usr/bin/whoami
	if err != nil {
		return fmt.Errorf("failed copying stdin/out/err: %s", err)
	}
	return
}

// nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(pushCmd)

	pushCmd.PersistentFlags().StringP("image", "i", "", "image name and version <image:version>")
	_ = pushCmd.MarkPersistentFlagRequired("image")
	_ = viper.BindPFlag("image", pushCmd.PersistentFlags().Lookup("image"))
	pushCmd.PersistentFlags().StringP("dest", "d", "", "remote destination <user@host>")
	_ = pushCmd.MarkPersistentFlagRequired("dest")
	_ = viper.BindPFlag("dest", pushCmd.PersistentFlags().Lookup("dest"))
	pushCmd.PersistentFlags().StringP("local", "l", "/tmp", "local temporary directory")
	_ = viper.BindPFlag("local", pushCmd.PersistentFlags().Lookup("local"))
	pushCmd.PersistentFlags().StringP("remote", "r", "/tmp", "remote temporary directory")
	_ = viper.BindPFlag("remote", pushCmd.PersistentFlags().Lookup("remote"))
	pushCmd.PersistentFlags().StringP("password", "p", "", "ssh password")
	_ = viper.BindPFlag("password", pushCmd.PersistentFlags().Lookup("password"))
	pushCmd.PersistentFlags().IntP("timeout", "t", 15, "scp timeout in minutes")
	_ = viper.BindPFlag("timeout", pushCmd.PersistentFlags().Lookup("timeout"))
	pushCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/ssh-docker-image.yaml)")
}
