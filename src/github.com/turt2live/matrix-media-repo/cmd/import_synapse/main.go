package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/howeyc/gopass"
	"github.com/sirupsen/logrus"
	"github.com/turt2live/matrix-media-repo/config"
	"github.com/turt2live/matrix-media-repo/logging"
	"github.com/turt2live/matrix-media-repo/rcontext"
	"github.com/turt2live/matrix-media-repo/services"
	"github.com/turt2live/matrix-media-repo/storage"
	"github.com/turt2live/matrix-media-repo/synapse"
)

func main() {
	postgresHost := flag.String("dbHost", "localhost", "The IP or hostname of the postgresql server with the synapse database")
	postgresPort := flag.Int("dbPort", 5432, "The port to access postgres on")
	postgresUsername := flag.String("dbUsername", "synapse", "The username to access postgres with")
	postgresPassword := flag.String("dbPassword", "", "The password to authorize the postgres user. Can be omitted to be prompted when run")
	postgresDatabase := flag.String("dbName", "synapse", "The name of the synapse database")
	baseUrl := flag.String("baseUrl", "http://localhost:8008", "The base URL to access your homeserver with")
	serverName := flag.String("serverName", "localhost", "The name of your homeserver (eg: matrix.org)")
	flag.Parse()

	var realPsqlPassword string
	if *postgresPassword == "" {
		fmt.Printf("Postgres password: ")
		pass, err := gopass.GetPasswd()
		if err != nil {
			panic(err)
		}
		realPsqlPassword = string(pass[:])
	} else {
		realPsqlPassword = *postgresPassword
	}

	c, err := config.ReadConfig()
	if err != nil {
		panic(err)
	}

	err = logging.Setup(c.General.LogDirectory)
	if err != nil {
		panic(err)
	}

	logrus.Info("Setting up for importing...")

	db, err := storage.OpenDatabase(c.Database.Postgres)
	if err != nil {
		panic(err)
	}

	connectionString := "postgres://" + *postgresUsername + ":" + realPsqlPassword + "@" + *postgresHost + ":" + strconv.Itoa(*postgresPort) + "/" + *postgresDatabase + "?sslmode=disable"
	csApiUrl := *baseUrl
	if csApiUrl[len(csApiUrl)-1:] == "/" {
		csApiUrl = csApiUrl[:len(csApiUrl)-1]
	}

	logrus.Info("Connecting to synapse database...")
	synDb, err := synapse.OpenDatabase(connectionString)
	if err != nil {
		panic(err)
	}

	logrus.Info("Fetching all local media records from synapse...")
	records, err := synDb.GetAllMedia()
	if err != nil {
		panic(err)
	}

	logrus.Info(fmt.Sprintf("Downloading %d media records", len(records)))
	ctx := context.TODO()
	for i := 0; i < len(records); i++ {
		percent := int((float32(i+1) / float32(len(records))) * 100)
		record := records[i]

		info := rcontext.RequestInfo{
			Log: logrus.WithFields(logrus.Fields{
				"mediaId": record.MediaId,
			}),
			Context: ctx,
			Db:      *db,
			Config:  c,
		}

		info.Log.Info(fmt.Sprintf("Downloading %s (%d/%d %d%%)", record.MediaId, i+1, len(records), percent))

		body, err := downloadMedia(csApiUrl, *serverName, record.MediaId)
		if err != nil {
			info.Log.Error(err.Error())
			continue
		}

		svc := services.CreateMediaService(info)

		_, err = svc.StoreMedia(body, record.ContentType, record.UploadName, record.UserId, *serverName, record.MediaId)
		if err != nil {
			info.Log.Error(err.Error())
			continue
		}

		body.Close()
	}

	logrus.Info("Import completed")
}

func downloadMedia(baseUrl string, serverName string, mediaId string) (io.ReadCloser, error) {
	downloadUrl := baseUrl + "/_matrix/media/r0/download/" + serverName + "/" + mediaId
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("received status code " + strconv.Itoa(resp.StatusCode))
	}

	return resp.Body, nil
}