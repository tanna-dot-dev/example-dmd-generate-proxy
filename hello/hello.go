package hello

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"encore.dev/beta/auth"
	"encore.dev/beta/errs"
	"encore.dev/pubsub"
	"encore.dev/rlog"
	"encore.dev/storage/sqldb"
)

//encore:authhandler
func AuthHandler(ctx context.Context, token string) (auth.UID, error) {
	// Validate the token and look up the user id,
	// for example by calling Firebase Auth.

	if token == "me" {
		return auth.UID("me"), nil
	}

	return "", &errs.Error{
		Code:    errs.Unauthenticated,
		Message: "invalid token",
	}
}

// TODO
//
//encore:api auth raw method=DELETE path=/packages.ecosyste.ms/*path
func DeleteCachedPackagesEcosystemsURL(w http.ResponseWriter, req *http.Request) {
	// TODO

	// then, asynchronously fetch the data
}

// ProxyPackagesEcosystems proxies a request to packages.ecosyste.ms, with caching in our PostgreSQL database
//
// TODO: auth
// TODO: rename method for GET
//
//encore:api auth raw method=GET path=/packages.ecosyste.ms/*path
func ProxyPackagesEcosystems(w http.ResponseWriter, req *http.Request) {
	upstreamURL, _ := strings.CutPrefix(req.URL.String(), "/packages.ecosyste.ms/")
	upstreamURL = "https://packages.ecosyste.ms/" + upstreamURL

	var data struct {
		Data json.RawMessage
	}

	// TODO: ~3s timeout on context?
	err := packagesEcosystemsResponsesDB.QueryRow(req.Context(), "SELECT data from packages_ecosystems_responses where url = $1", upstreamURL).Scan(&data.Data)
	if errors.Is(err, sqldb.ErrNoRows) {
		// ignore, as then we'll then get it fresh
	} else if err != nil {
		rlog.Warn("Failed to look up whether we had a ", "err", err)
	}

	// TODO: if cached response is older than 1 week? day?, then bypass

	if data.Data != nil {
		rlog.Debug("Returning cached response", "upstreamURL", upstreamURL, "numBytes", len(data.Data))
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write(data.Data)
		return
	}

	// TODO: understand what should and shoudln't be cached i.e. a response of `[]` from a pURL lookup

	// TODO shared client
	resp, err := http.DefaultClient.Get(upstreamURL)
	if err != nil {
		rlog.Warn("Failed to send request upstream to Ecosystems", "err", err, "upstreamURL", upstreamURL)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// TODO: respect `Cache-Control`

	if resp.StatusCode == http.StatusNotFound {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if resp.StatusCode != http.StatusOK {
		rlog.Warn(fmt.Sprintf("Ecosystems responded with an HTTP %d, returning HTTP 500 to caller", resp.StatusCode), "err", err, "upstreamURL", upstreamURL)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		rlog.Warn("Failed to retrieve response body from Ecosystems", "err", err, "upstreamURL", upstreamURL)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("content-type", "application/json")
	_, _ = w.Write(respBytes)

	go func(url string, data []byte) {

		// because we're now in a new context, the request has ended
		id, err := WritePackagesEcosystemsResponseEvents.Publish(context.Background(), &WritePackagesEcosystemsResponseEvent{
			URL:         upstreamURL,
			Data:        respBytes,
			LastUpdated: time.Now(),
		})
		if err != nil {
			rlog.Error("Failed to publish WritePackagesEcosystemsResponseEvent", "upstreamURL", upstreamURL, "err", err)
			return
		}

		rlog.Debug("Published WritePackagesEcosystemsResponseEvent", "upstreamURL", upstreamURL, "eventID", id)
	}(upstreamURL, respBytes)

}

type WritePackagesEcosystemsResponseEvent struct {
	URL         string
	Data        []byte
	LastUpdated time.Time
}

var WritePackagesEcosystemsResponseEvents = pubsub.NewTopic[*WritePackagesEcosystemsResponseEvent]("write-package-ecosystems-response-events", pubsub.TopicConfig{
	DeliveryGuarantee: pubsub.AtLeastOnce,
})

var packagesEcosystemsResponsesDB = sqldb.NewDatabase("packages_ecosystems_responses", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})

var _ = pubsub.NewSubscription(WritePackagesEcosystemsResponseEvents, "write-package-ecosystems-response-events", pubsub.SubscriptionConfig[*WritePackagesEcosystemsResponseEvent]{
	Handler: HandleWritePackagesEcosystemsResponseEvent,
})

func HandleWritePackagesEcosystemsResponseEvent(ctx context.Context, msg *WritePackagesEcosystemsResponseEvent) error {
	_, err := packagesEcosystemsResponsesDB.Exec(ctx, `
	insert into packages_ecosystems_responses (url, data, last_updated) VALUES ($1, $2, $3) on conflict (url) do update set data = $2, last_updated = $3
	`, msg.URL, string(msg.Data), msg.LastUpdated.Format(time.RFC3339))
	if err != nil {
		rlog.Error("Failed to persist data", "upstreamURL", msg.URL, "err", err)
		return err
	}
	rlog.Debug("Persisted data", "upstreamURL", msg.URL)

	return nil
}
