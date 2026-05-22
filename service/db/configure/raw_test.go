package configure

import "testing"

func TestBytes2SubscriptionRawWrappedServerRaw(t *testing.T) {
	data := []byte(`{"address":"x","status":"","servers":[{"Type":5,"Raw":"{\"serverObj\":{\"protocol\":\"vmess\",\"ps\":\"main-admin\",\"add\":\"127.0.0.1\",\"port\":\"443\",\"id\":\"11111111-1111-1111-1111-111111111111\",\"aid\":\"0\",\"net\":\"tcp\",\"type\":\"none\",\"host\":\"\",\"path\":\"\",\"tls\":\"tls\"},\"latency\":\"\"}"}]}`)
	s, err := Bytes2SubscriptionRaw(data)
	if err != nil {
		t.Fatalf("Bytes2SubscriptionRaw error: %v", err)
	}
	if len(s.Servers) != 1 || s.Servers[0].ServerObj == nil {
		t.Fatalf("server obj not parsed from wrapped Raw")
	}
}
