package tariff

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff/awattar"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

type Awattar struct {
	mux  sync.Mutex
	log  *util.Logger
	uri  string
	data api.Rates
}

var _ api.Tariff = (*Awattar)(nil)

func NewAwattar(other map[string]interface{}) (*Awattar, error) {
	cc := struct {
		Cheap  any // TODO deprecated
		Region string
	}{
		Region: "DE",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	t := &Awattar{
		log: util.NewLogger("awattar"),
		uri: fmt.Sprintf(awattar.RegionURI, strings.ToLower(cc.Region)),
	}

	// TODO deprecated
	if cc.Cheap != nil {
		t.log.WARN.Println("cheap rate configuration has been replaced by target charging and is deprecated")
	}

	go t.Run()

	return t, nil
}

func (t *Awattar) Run() {
	client := request.NewHelper(t.log)

	for ; true; <-time.NewTicker(time.Hour).C {
		var res awattar.Prices
		if err := client.GetJSON(t.uri, &res); err != nil {
			t.log.ERROR.Println(err)
			continue
		}

		t.mux.Lock()
		t.data = make(api.Rates, 0, len(res.Data))

		for _, r := range res.Data {
			ar := api.Rate{
				Start: r.StartTimestamp,
				End:   r.EndTimestamp,
				Price: r.Marketprice / 1e3,
			}
			t.data = append(t.data, ar)
		}

		t.mux.Unlock()
	}
}

// Rates implements the api.Tariff interface
func (t *Awattar) Rates() (api.Rates, error) {
	t.mux.Lock()
	defer t.mux.Unlock()
	return append([]api.Rate{}, t.data...), nil
}
