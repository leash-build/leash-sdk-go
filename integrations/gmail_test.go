package integrations

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestGmail_ListMessages(t *testing.T) {
	stub := newStub(map[string]stubResponse{
		"gmail/list-messages": okJSON(`{"messages":[{"id":"m1","threadId":"t1"}],"resultSizeEstimate":1}`),
	})
	g := New(stub).Gmail()

	got, err := g.ListMessages(context.Background(), &GmailListParams{MaxResults: 5, Query: "from:x"})
	if err != nil {
		t.Fatal(err)
	}
	want := &GmailMessageList{
		Messages:           []GmailMessage{{ID: "m1", ThreadID: "t1"}},
		ResultSizeEstimate: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	call := stub.Calls[0]
	body, _ := json.Marshal(call.Body)
	if string(body) != `{"query":"from:x","maxResults":5}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestGmail_ListMessages_NilParams(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/list-messages": okJSON(`{}`)})
	if _, err := New(stub).Gmail().ListMessages(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if stub.Calls[0].Body != nil {
		t.Errorf("expected nil body for nil params, got %v", stub.Calls[0].Body)
	}
}

func TestGmail_GetMessage(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/get-message": okJSON(`{"id":"abc"}`)})
	g := New(stub).Gmail()
	raw, err := g.GetMessage(context.Background(), "abc", GmailFormatMetadata)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"id":"abc"}` {
		t.Errorf("unexpected raw: %s", raw)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"format":"metadata","messageId":"abc"}` {
		t.Errorf("unexpected request body: %s", body)
	}
}

func TestGmail_SendMessage(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/send-message": okJSON(`{"id":"sent"}`)})
	g := New(stub).Gmail()
	_, err := g.SendMessage(context.Background(), GmailSendMessageParams{To: "a@b.c", Subject: "hi", Body: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"to":"a@b.c","subject":"hi","body":"hello"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestGmail_SearchMessages(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/search-messages": okJSON(`{"messages":[]}`)})
	if _, err := New(stub).Gmail().SearchMessages(context.Background(), "q", 10); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(stub.Calls[0].Body)
	if string(body) != `{"maxResults":10,"query":"q"}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestGmail_ListLabels(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/list-labels": okJSON(`{"labels":[{"id":"INBOX","name":"Inbox","type":"system"}]}`)})
	got, err := New(stub).Gmail().ListLabels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Labels) != 1 || got.Labels[0].ID != "INBOX" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestGmail_GetProfile(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/get-profile": okJSON(`{"emailAddress":"x@y.z"}`)})
	raw, err := New(stub).Gmail().GetProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"emailAddress":"x@y.z"}` {
		t.Errorf("unexpected raw: %s", raw)
	}
}

func TestGmail_PropagatesError(t *testing.T) {
	stub := newStub(map[string]stubResponse{"gmail/list-messages": {err: &stubError{msg: "boom"}}})
	if _, err := New(stub).Gmail().ListMessages(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}
