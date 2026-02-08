package handlers

import (
	"bytes"
	"encoding/csv"
	"net"
	"net/http"
	"strconv"

	"github.com/JakeNeyer/ipam/store"
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

// ExportCSVHandler returns an http.HandlerFunc that writes network blocks as CSV.
func ExportCSVHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		envs, err := s.ListEnvironments()
		if err != nil {
			http.Error(w, "failed to list environments", http.StatusInternalServerError)
			return
		}
		blocks, err := s.ListBlocks()
		if err != nil {
			http.Error(w, "failed to list blocks", http.StatusInternalServerError)
			return
		}

		envByID := make(map[string]string)
		for _, e := range envs {
			envByID[e.Id.String()] = e.Name
		}

		var buf bytes.Buffer
		wr := csv.NewWriter(&buf)
		if err := wr.Write([]string{"name", "cidr", "cidr_start", "cidr_end", "environment_name", "total_ips", "used_ips", "available_ips"}); err != nil {
			http.Error(w, "failed to write CSV", http.StatusInternalServerError)
			return
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
			http.Error(w, "failed to write CSV", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ipam-export.csv"`)
		_, _ = w.Write(buf.Bytes())
	}
}
