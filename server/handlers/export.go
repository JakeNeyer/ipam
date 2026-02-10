package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	"github.com/JakeNeyer/ipam/store"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
)

// cidrStartEnd returns the first and last IP of a CIDR as strings, or "", "" if invalid.
func cidrStartEnd(cidr string) (start, end string) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", ""
	}
	first := make(net.IP, len(n.IP))
	copy(first, n.IP)
	last := make(net.IP, len(n.IP))
	copy(last, n.IP)
	for i := range last {
		last[i] = last[i] | (n.Mask[i] ^ 0xff)
	}
	return first.String(), last.String()
}

// exportCSVOutput is the output for GET /api/export/csv. Body is written as text/csv by the CSV response encoder.
type exportCSVOutput struct {
	Body []byte
}

// NewExportCSVUseCase returns a use case for GET /api/export/csv.
func NewExportCSVUseCase(s store.Storer) usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *exportCSVOutput) error {
		envs, err := s.ListEnvironments()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}
		blocks, err := s.ListBlocks()
		if err != nil {
			return status.Wrap(err, status.Internal)
		}

		envByID := make(map[string]string)
		for _, e := range envs {
			envByID[e.Id.String()] = e.Name
		}

		var buf bytes.Buffer
		wr := csv.NewWriter(&buf)
		if err := wr.Write([]string{"name", "cidr", "cidr_start", "cidr_end", "environment_name", "total_ips", "used_ips", "available_ips"}); err != nil {
			return status.Wrap(err, status.Internal)
		}

		for _, b := range blocks {
			used := computeUsedIPsForBlock(s, b.Name)
			avail := b.Usage.TotalIPs - used
			if avail < 0 {
				avail = 0
			}
			envName := envByID[b.EnvironmentID.String()]
			start, end := cidrStartEnd(b.CIDR)
			_ = wr.Write([]string{
				b.Name,
				b.CIDR,
				start,
				end,
				envName,
				strconv.Itoa(b.Usage.TotalIPs),
				strconv.Itoa(used),
				strconv.Itoa(avail),
			})
		}

		wr.Flush()
		if wr.Error() != nil {
			return status.Wrap(errors.New("failed to write CSV"), status.Internal)
		}

		output.Body = buf.Bytes()
		return nil
	})
	u.SetTitle("Export CSV")
	u.SetDescription("Exports all blocks as CSV")
	u.SetExpectedErrors(status.Internal)
	return u
}

// csvResponseEncoder writes exportCSVOutput as text/csv.
type csvResponseEncoder struct{}

// NewCSVResponseEncoder returns a ResponseEncoder that writes exportCSVOutput.Body as text/csv.
func NewCSVResponseEncoder() nethttp.ResponseEncoder {
	return &csvResponseEncoder{}
}

func (e *csvResponseEncoder) WriteErrResponse(w http.ResponseWriter, _ *http.Request, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(response)
}

func (e *csvResponseEncoder) WriteSuccessfulResponse(w http.ResponseWriter, _ *http.Request, output interface{}, _ rest.HandlerTrait) {
	out, ok := output.(*exportCSVOutput)
	if !ok || out == nil {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ipam-export.csv"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out.Body)
}

func (e *csvResponseEncoder) SetupOutput(output interface{}, _ *rest.HandlerTrait) {}

func (e *csvResponseEncoder) MakeOutput(w http.ResponseWriter, _ rest.HandlerTrait) interface{} {
	return &exportCSVOutput{}
}
