package kubecost

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// queryDecorator adds a key and value to the request URL.
func queryDecorator(key, val string) Decorator {
	return func(request *http.Request) {
		query := request.URL.Query()
		query.Add(key, val)

		request.URL.RawQuery = query.Encode()
	}
}

// Window adds the window parameter to a query.
func Window(window string) Decorator {
	return queryDecorator("window", window)
}

// DateWindow adds window parameter in the format of start,end to a query.
func DateWindow(start, end time.Time) Decorator {
	return queryDecorator("window", strings.Join([]string{
		start.UTC().Format(time.RFC3339),
		end.UTC().Format(time.RFC3339),
	}, ","))
}

// Aggregate adds the aggregate parameter to a query.
func Aggregate(aggregates []string) Decorator {
	return queryDecorator("aggregate", strings.Join(aggregates, ","))
}

// Accumulate adds the accumalate argument to a query.
func Accumulate(accumulate bool) Decorator {
	return queryDecorator("accumulate", strconv.FormatBool(accumulate))
}

// Idle adds the idle argument to a query.
func Idle(idle bool) Decorator {
	return queryDecorator("idle", strconv.FormatBool(idle))
}

// External adds the external argument to a query.
func External(val bool) Decorator {
	return queryDecorator("external", strconv.FormatBool(val))
}

// ShareIdle adds the shareIdle argument to a query.
func ShareIdle(val bool) Decorator {
	return queryDecorator("shareIdle", strconv.FormatBool(val))
}

// ShareTenancyCosts adds teh shareTenancyCosts argument to a query.
func ShareTenancyCosts(val bool) Decorator {
	return queryDecorator("shareTenancyCosts", strconv.FormatBool(val))
}
