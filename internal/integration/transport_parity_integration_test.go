//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientapp "github.com/xhrobj/gopherkeeper/internal/client/app"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestIntegration_HTTPSAndGRPCTransportParityFlow(t *testing.T) {
	fixture := newDualTransportFixture(t)
	httpsRuntime := fixture.newRuntime(t, config.TransportHTTPS)
	grpcRuntime := fixture.newRuntime(t, config.TransportGRPC)

	assertTransportHealth(t, fixture.ctx, httpsRuntime, "HTTPS")
	assertTransportHealth(t, fixture.ctx, grpcRuntime, "gRPC")

	registerThroughGRPCAndLoginThroughHTTPSCLI(t, fixture)

	httpsUser, err := httpsRuntime.Whoami(fixture.ctx)
	if err != nil {
		t.Fatalf("read CLI-created session through HTTPS application: %v", err)
	}
	grpcUser, err := grpcRuntime.Whoami(fixture.ctx)
	if err != nil {
		t.Fatalf("reuse HTTPS CLI session through gRPC: %v", err)
	}
	assertSameUser(t, grpcUser, httpsUser, "gRPC current user")

	records := createParityRecords(t, fixture.ctx, httpsRuntime, grpcRuntime)
	assertSameRecordList(t, fixture.ctx, httpsRuntime, grpcRuntime, records)
	updateParityRecords(t, fixture.ctx, httpsRuntime, grpcRuntime, records)
	records[model.RecordTypeText] = assertCrossTransportConflict(
		t,
		fixture.ctx,
		httpsRuntime,
		grpcRuntime,
		records[model.RecordTypeText],
	)
	assertInitialGRPCSync(t, fixture, httpsRuntime, grpcRuntime)
	assertCrossTransportDelete(t, fixture.ctx, httpsRuntime, grpcRuntime, records[model.RecordTypeBinary])
	records[model.RecordTypeText] = assertGRPCSyncRefreshAndOfflineRead(
		t,
		fixture,
		httpsRuntime,
		grpcRuntime,
		records,
	)
	deleteRemainingParityRecords(t, fixture.ctx, httpsRuntime, grpcRuntime, records)
}

func TestIntegration_GRPCTransportDoesNotFallbackToHTTPS(t *testing.T) {
	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	httpsAddress, stopHTTPS := startHTTPSServer(
		t,
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
				t.Errorf("write HTTPS health response: %v", err)
			}
		}),
		serverCertFile,
		serverKeyFile,
	)
	t.Cleanup(stopHTTPS)

	baseConfig := config.Config{
		Address:     httpsAddress,
		GRPCAddress: reserveClosedAddress(t),
		CACertFile:  caCertFile,
		SessionDir:  t.TempDir(),
		CacheDir:    t.TempDir(),
	}

	httpsConfig := baseConfig
	httpsConfig.Transport = config.TransportHTTPS
	httpsRuntime, err := clientapp.NewRuntime(httpsConfig)
	if err != nil {
		t.Fatalf("create HTTPS runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := httpsRuntime.Close(); err != nil {
			t.Errorf("close HTTPS runtime: %v", err)
		}
	})
	if status, err := httpsRuntime.Health(context.Background()); err != nil || status != "ok" {
		t.Fatalf("control HTTPS health = %q, error %v; want ok", status, err)
	}

	grpcConfig := baseConfig
	grpcConfig.Transport = config.TransportGRPC
	grpcRuntime, err := clientapp.NewRuntime(grpcConfig)
	if err != nil {
		t.Fatalf("create gRPC runtime: %v", err)
	}
	t.Cleanup(func() {
		if err := grpcRuntime.Close(); err != nil {
			t.Errorf("close gRPC runtime: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	status, err := grpcRuntime.Health(ctx)
	if err == nil {
		t.Fatalf("gRPC health with closed gRPC address = %q, want error instead of HTTPS fallback", status)
	}
	if status != "" {
		t.Fatalf("gRPC health status = %q, want empty on connection error", status)
	}
}

func registerThroughGRPCAndLoginThroughHTTPSCLI(t *testing.T, fixture dualTransportFixture) {
	t.Helper()

	stdout, stderr, err := runDualTransportCLICommand(
		fixture,
		config.TransportGRPC,
		[]string{"register", "--login", " Alice "},
		testRegistrationPassword+"\n"+testRegistrationPassword+"\n",
	)
	if err != nil {
		t.Fatalf("register Alice through gRPC CLI: %v", err)
	}
	if stdout != "User alice registered successfully.\n" || stderr != "" {
		t.Fatalf("gRPC CLI registration output = %q, stderr = %q", stdout, stderr)
	}

	stdout, stderr, err = runDualTransportCLICommand(
		fixture,
		config.TransportHTTPS,
		[]string{"login", "--login", "ALICE"},
		testRegistrationPassword+"\n",
	)
	if err != nil {
		t.Fatalf("login Alice through HTTPS CLI: %v", err)
	}
	if stdout != "User alice logged in successfully.\n" || stderr != "" {
		t.Fatalf("HTTPS CLI login output = %q, stderr = %q", stdout, stderr)
	}

	stdout, stderr, err = runDualTransportCLICommand(
		fixture,
		config.TransportGRPC,
		[]string{"whoami"},
		"",
	)
	if err != nil {
		t.Fatalf("reuse HTTPS session through gRPC CLI: %v", err)
	}
	if stdout != "alice\n" || stderr != "" {
		t.Fatalf("gRPC CLI whoami output = %q, stderr = %q", stdout, stderr)
	}
}

func runDualTransportCLICommand(
	fixture dualTransportFixture,
	transport config.Transport,
	command []string,
	input string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--transport", string(transport),
		"--address", fixture.httpsAddress,
		"--grpc-address", fixture.grpcAddress,
		"--ca-cert", fixture.caCertFile,
		"--session-dir", fixture.sessionDir,
		"--cache-dir", fixture.cacheDir,
	}
	args = append(args, command...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := clientcli.RunWithInput(
		fixture.ctx,
		args,
		strings.NewReader(input),
		&stdout,
		&stderr,
		buildinfo.Info{},
	)

	return stdout.String(), stderr.String(), err
}

func assertTransportHealth(
	t *testing.T,
	ctx context.Context,
	runtime *clientapp.Runtime,
	transport string,
) {
	t.Helper()

	status, err := runtime.Health(ctx)
	if err != nil {
		t.Fatalf("%s health: %v", transport, err)
	}
	if status != "ok" {
		t.Fatalf("%s health = %q, want ok", transport, status)
	}
}

func createParityRecords(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
) map[model.RecordType]model.Record {
	t.Helper()

	month := 12
	year := 30
	cases := []struct {
		title   string
		payload model.RecordPayload
		runtime *clientapp.Runtime
	}{
		{
			title: "Alice credentials",
			payload: &model.CredentialsPayload{
				Login:    "alice@example.com",
				Password: "initial-credentials-secret",
				URL:      "https://example.com",
				Metadata: "credentials metadata",
			},
			runtime: httpsRuntime,
		},
		{
			title: "Alice card",
			payload: &model.CardPayload{
				Number:      "4111111111111111",
				Cardholder:  "JOEL MILLER",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "014",
				Metadata:    "card metadata",
			},
			runtime: grpcRuntime,
		},
		{
			title: "Alice note",
			payload: &model.TextPayload{
				Text:     "initial text secret",
				Metadata: "text metadata",
			},
			runtime: httpsRuntime,
		},
		{
			title: "Alice binary",
			payload: &model.BinaryPayload{
				Filename: "secret.bin",
				Data:     []byte{0x00, 0x01, 0x7f, 0xff},
				Metadata: "binary metadata",
			},
			runtime: grpcRuntime,
		},
	}

	created := make(map[model.RecordType]model.Record, len(cases))
	for _, test := range cases {
		record, err := test.runtime.CreateRecord(ctx, usecase.CreateRecordRequest{
			Title:   test.title,
			Payload: test.payload,
		})
		if err != nil {
			t.Fatalf("create %s record: %v", test.payload.RecordType(), err)
		}
		if record.Metadata.Revision != model.RecordInitialRevision {
			t.Fatalf("created %s revision = %d, want 1", test.payload.RecordType(), record.Metadata.Revision)
		}
		assertParityRecord(t, record, test.title, model.RecordInitialRevision, test.payload)
		created[test.payload.RecordType()] = record
	}

	return created
}

func assertSameRecordList(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	expected map[model.RecordType]model.Record,
) {
	t.Helper()

	httpsRecords, err := httpsRuntime.ListRecords(ctx)
	if err != nil {
		t.Fatalf("list records through HTTPS: %v", err)
	}
	grpcRecords, err := grpcRuntime.ListRecords(ctx)
	if err != nil {
		t.Fatalf("list records through gRPC: %v", err)
	}

	if len(httpsRecords) != len(expected) || len(grpcRecords) != len(expected) {
		t.Fatalf(
			"listed record counts = HTTPS %d, gRPC %d, want %d",
			len(httpsRecords),
			len(grpcRecords),
			len(expected),
		)
	}

	sort.Slice(httpsRecords, func(i, j int) bool { return httpsRecords[i].ID < httpsRecords[j].ID })
	sort.Slice(grpcRecords, func(i, j int) bool { return grpcRecords[i].ID < grpcRecords[j].ID })
	for index := range httpsRecords {
		assertSameRecordMetadata(t, httpsRecords[index], grpcRecords[index])
	}
}

func updateParityRecords(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	records map[model.RecordType]model.Record,
) {
	t.Helper()

	month := 1
	year := 31
	updates := []struct {
		recordType model.RecordType
		title      string
		payload    model.RecordPayload
		runtime    *clientapp.Runtime
		reader     *clientapp.Runtime
	}{
		{
			recordType: model.RecordTypeCredentials,
			title:      "Updated Alice credentials",
			payload: &model.CredentialsPayload{
				Login:    "updated@example.com",
				Password: "updated-credentials-secret",
				URL:      "https://updated.example.com",
				Metadata: "updated credentials metadata",
			},
			runtime: grpcRuntime,
			reader:  httpsRuntime,
		},
		{
			recordType: model.RecordTypeCard,
			title:      "Updated Alice card",
			payload: &model.CardPayload{
				Number:      "5555555555554444",
				Cardholder:  "ALICE",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "123",
				Metadata:    "updated card metadata",
			},
			runtime: httpsRuntime,
			reader:  grpcRuntime,
		},
		{
			recordType: model.RecordTypeText,
			title:      "Updated Alice note",
			payload: &model.TextPayload{
				Text:     "updated text secret",
				Metadata: "updated text metadata",
			},
			runtime: grpcRuntime,
			reader:  httpsRuntime,
		},
		{
			recordType: model.RecordTypeBinary,
			title:      "Updated Alice binary",
			payload: &model.BinaryPayload{
				Filename: "updated.bin",
				Data:     []byte{0xde, 0xad, 0xbe, 0xef},
				Metadata: "updated binary metadata",
			},
			runtime: httpsRuntime,
			reader:  grpcRuntime,
		},
	}

	for _, update := range updates {
		initial := records[update.recordType]
		updated, err := update.runtime.UpdateRecord(ctx, usecase.UpdateRecordRequest{
			RecordID:         initial.Metadata.ID,
			ExpectedRevision: initial.Metadata.Revision,
			Title:            update.title,
			Payload:          update.payload,
		})
		if err != nil {
			t.Fatalf("update %s record: %v", update.recordType, err)
		}
		assertParityRecord(t, updated, update.title, 2, update.payload)

		read, err := update.reader.GetRecord(ctx, initial.Metadata.ID)
		if err != nil {
			t.Fatalf("read updated %s record through opposite transport: %v", update.recordType, err)
		}
		assertParityRecord(t, read, update.title, 2, update.payload)
		records[update.recordType] = updated
	}
}

func assertCrossTransportConflict(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	current model.Record,
) model.Record {
	t.Helper()

	updated, err := httpsRuntime.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID:         current.Metadata.ID,
		ExpectedRevision: current.Metadata.Revision,
		Title:            "Latest Alice note",
		Payload: &model.TextPayload{
			Text:     "latest text secret",
			Metadata: "latest text metadata",
		},
	})
	if err != nil {
		t.Fatalf("latest HTTPS text update: %v", err)
	}

	_, err = grpcRuntime.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID:         current.Metadata.ID,
		ExpectedRevision: current.Metadata.Revision,
		Title:            "Stale Alice note",
		Payload: &model.TextPayload{
			Text: "stale text secret",
		},
	})
	if !errors.Is(err, model.ErrRecordRevisionConflict) {
		t.Fatalf("stale gRPC update error = %v, want record revision conflict", err)
	}

	return updated
}

func assertCrossTransportDelete(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	record model.Record,
) {
	t.Helper()

	if err := httpsRuntime.DeleteRecord(ctx, usecase.DeleteRecordRequest{
		RecordID:         record.Metadata.ID,
		ExpectedRevision: record.Metadata.Revision,
	}); err != nil {
		t.Fatalf("delete binary through HTTPS: %v", err)
	}

	_, err := grpcRuntime.GetRecord(ctx, record.Metadata.ID)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("get deleted binary through gRPC error = %v, want record not found", err)
	}
}

func assertInitialGRPCSync(
	t *testing.T,
	fixture dualTransportFixture,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
) {
	t.Helper()

	result, err := grpcRuntime.Sync(fixture.ctx, usecase.SyncRequest{Password: testRegistrationPassword})
	if err != nil {
		t.Fatalf("initial sync through gRPC: %v", err)
	}
	if len(result.Added) != 4 ||
		len(result.Updated) != 0 ||
		len(result.Removed) != 0 ||
		result.Unchanged != 0 {
		t.Fatalf("initial gRPC sync = %#v, want 4 added", result)
	}

	whoami, err := httpsRuntime.Whoami(fixture.ctx)
	if err != nil {
		t.Fatalf("reuse gRPC-refreshed session through HTTPS: %v", err)
	}
	if whoami.Login != "alice" {
		t.Fatalf("HTTPS current user after gRPC sync = %q, want alice", whoami.Login)
	}
}

func assertGRPCSyncRefreshAndOfflineRead(
	t *testing.T,
	fixture dualTransportFixture,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	records map[model.RecordType]model.Record,
) model.Record {
	t.Helper()

	textRecord, err := httpsRuntime.GetRecord(fixture.ctx, records[model.RecordTypeText].Metadata.ID)
	if err != nil {
		t.Fatalf("get current text before refresh sync: %v", err)
	}
	latestPayload := &model.TextPayload{
		Text:     "cache refresh through gRPC",
		Metadata: "cache refresh metadata",
	}
	latest, err := httpsRuntime.UpdateRecord(fixture.ctx, usecase.UpdateRecordRequest{
		RecordID:         textRecord.Metadata.ID,
		ExpectedRevision: textRecord.Metadata.Revision,
		Title:            "Cache refresh note",
		Payload:          latestPayload,
	})
	if err != nil {
		t.Fatalf("update text through HTTPS before gRPC sync: %v", err)
	}

	result, err := grpcRuntime.Sync(fixture.ctx, usecase.SyncRequest{Password: testRegistrationPassword})
	if err != nil {
		t.Fatalf("refresh sync through gRPC: %v", err)
	}
	if len(result.Added) != 0 ||
		len(result.Updated) != 1 ||
		len(result.Removed) != 1 ||
		result.Unchanged != 2 {
		t.Fatalf("refresh gRPC sync = %#v, want 1 updated, 1 removed, and 2 unchanged", result)
	}

	offline := clientapp.NewOffline(fixture.clientConfig(config.TransportGRPC))
	request := usecase.OfflineReadRequest{Login: "ALICE", Password: testRegistrationPassword}
	cached, err := offline.GetCachedRecord(fixture.ctx, request, latest.Metadata.ID)
	if err != nil {
		t.Fatalf("read gRPC-synchronized record offline: %v", err)
	}
	if !cached.MayBeStale || cached.Source != usecase.OfflineSourceLocalCache {
		t.Fatalf("offline source = %q stale=%t, want local cache and stale warning", cached.Source, cached.MayBeStale)
	}
	assertParityRecord(t, cached.Record, latest.Metadata.Title, latest.Metadata.Revision, latestPayload)

	_, err = offline.GetCachedRecord(
		fixture.ctx,
		request,
		records[model.RecordTypeBinary].Metadata.ID,
	)
	if !errors.Is(err, usecase.ErrCachedRecordNotFound) {
		t.Fatalf("offline deleted binary error = %v, want cached record not found", err)
	}

	return latest
}

func deleteRemainingParityRecords(
	t *testing.T,
	ctx context.Context,
	httpsRuntime *clientapp.Runtime,
	grpcRuntime *clientapp.Runtime,
	records map[model.RecordType]model.Record,
) {
	t.Helper()

	deletions := []struct {
		record  model.Record
		runtime *clientapp.Runtime
	}{
		{record: records[model.RecordTypeCredentials], runtime: grpcRuntime},
		{record: records[model.RecordTypeCard], runtime: httpsRuntime},
		{record: records[model.RecordTypeText], runtime: grpcRuntime},
	}

	for _, deletion := range deletions {
		if err := deletion.runtime.DeleteRecord(ctx, usecase.DeleteRecordRequest{
			RecordID:         deletion.record.Metadata.ID,
			ExpectedRevision: deletion.record.Metadata.Revision,
		}); err != nil {
			t.Fatalf("delete %s record: %v", deletion.record.Metadata.Type, err)
		}
	}

	for name, runtime := range map[string]*clientapp.Runtime{
		"HTTPS": httpsRuntime,
		"gRPC":  grpcRuntime,
	} {
		listed, err := runtime.ListRecords(ctx)
		if err != nil {
			t.Fatalf("list records through %s after deletes: %v", name, err)
		}
		if len(listed) != 0 {
			t.Fatalf("%s record count after deletes = %d, want 0", name, len(listed))
		}
	}
}

func assertSameUser(t *testing.T, got, want model.User, description string) {
	t.Helper()

	if got.ID != want.ID || got.Login != want.Login || !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("%s = %#v, want %#v", description, got, want)
	}
}

func assertSameRecordMetadata(t *testing.T, got, want model.RecordMetadata) {
	t.Helper()

	if got.ID != want.ID ||
		got.Type != want.Type ||
		got.Title != want.Title ||
		got.Revision != want.Revision ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Fatalf("record metadata = %#v, want %#v", got, want)
	}
}

func assertParityRecord(
	t *testing.T,
	record model.Record,
	title string,
	revision int64,
	payload model.RecordPayload,
) {
	t.Helper()

	if record.Metadata.Title != title || record.Metadata.Revision != revision {
		t.Fatalf(
			"record metadata = title %q revision %d, want title %q revision %d",
			record.Metadata.Title,
			record.Metadata.Revision,
			title,
			revision,
		)
	}
	if record.Metadata.Type != payload.RecordType() {
		t.Fatalf("record type = %q, want %q", record.Metadata.Type, payload.RecordType())
	}
	if !reflect.DeepEqual(record.Payload, payload) {
		t.Fatalf("record payload = %#v, want %#v", record.Payload, payload)
	}
}
