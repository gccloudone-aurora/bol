package report

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/gccloudone-aurora/bol/pkg/kubecost"
	"github.com/gccloudone-aurora/bol/pkg/kubernetes"
	"github.com/gccloudone-aurora/bol/pkg/storage"
	"github.com/gccloudone-aurora/bol/pkg/util"
	v1 "k8s.io/api/core/v1"
)

type ReportGenerator struct {
	StartDate      *time.Time
	EndDate        *time.Time
	FileNameSuffix string
	Storage        storage.Storage
	clusterData    map[string]clusterData
}

type clusterData struct {
	name           string
	subscription   string
	namespaces     map[string]v1.Namespace
	kubecostClient *kubecost.Client
}

func NewReport(ctx context.Context, config util.Config) (*ReportGenerator, error) {
	storage, err := storage.NewStorage(config.ArtifactRepository)
	if err != nil {
		log.Fatalf("NewReportGenerator: Error creating storage instance: %v", err)
	}

	// Determine what dates a report should be generated for
	start, end, err := daysToGenerateReport(ctx, storage, regexp.MustCompile(fmt.Sprintf("kubecost_%s%s.csv", "(\\d{4}-\\d{2}-\\d{2})", config.FileNameSuffix)), config.MaximumReportingWindowInDays)
	if err != nil {
		log.Fatalf("NewReportGenerator: Error %v", err)
	}

	clusters := make(map[string]clusterData)
	for _, cluster := range config.Clusters {
		kubernetesClientset, err := kubernetes.CreateClientset(cluster.KubernetesAuth)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
		namespaces, err := kubernetes.GetNamespaces(*kubernetesClientset)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		kubecostClient, err := kubecost.NewClient(util.MustParseURL(cluster.KubecostURL), cluster.KubecostURlAttachAzurebearerToken)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		clusters[cluster.Name] = clusterData{
			name:           cluster.Name,
			subscription:   cluster.Subscription,
			namespaces:     namespaces,
			kubecostClient: kubecostClient,
		}
	}

	return &ReportGenerator{
		StartDate:      start,
		EndDate:        end,
		FileNameSuffix: config.FileNameSuffix,
		Storage:        storage,
		clusterData:    clusters,
	}, nil
}

// daysToGenerateReport returns the start and end date to generate a report.
// It will find the last date we have data written in the storage account and
// generate a report from that date to the current date (assuming the difference is less than 30).
func daysToGenerateReport(ctx context.Context, storage storage.Storage, fileNameRegex *regexp.Regexp, maximumReportingWindowInDays int) (*time.Time, *time.Time, error) {
	lastDate, err := storage.LastDateOfFileUploaded(ctx, fileNameRegex)
	if err != nil {
		return nil, nil, fmt.Errorf("daysToGenerateReport: Failed determining the range of dates that a report should be generated for: %w", err)
	}

	// Kubecost has up to 30 days of data
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, (maximumReportingWindowInDays * -1))
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if lastDate != nil && lastDate.After(start) {
		start = lastDate.AddDate(0, 0, 1)
	}

	if start == end {
		return &start, &end, fmt.Errorf("daysToGenerateReport: A report has been created today. There is no new data to generate a report for.")
	}

	return &start, &end, nil
}

func (r *ReportGenerator) Generate(ctx context.Context) {
	for date := *r.StartDate; date.Before(*r.EndDate); date = date.AddDate(0, 0, 1) {
		log.Printf("\n\nGenerating report for %s", date)

		pipeReader, err := r.writeReport(ctx, date)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		reportName := fmt.Sprintf("kubecost_%s%s.csv", date.Format("2006-01-02"), r.FileNameSuffix)
		if err := r.Storage.UploadArtifact(ctx, reportName, pipeReader); err != nil {
			log.Fatalf("Error uploading report: %w \n", err)
		}
		log.Printf("Report %s uploaded successfully!\n\n", reportName)
	}
}

func (r *ReportGenerator) writeReport(ctx context.Context, date time.Time) (io.Reader, error) {
	pipeReader, pipeWriter := io.Pipe()
	csv := csv.NewWriter(pipeWriter)
	go func() {
		csv.Write([]string{"Cluster", "Subscription", "WorkloadID", "UserID", "Namespace", "Date", "CPUCost", "MemoryCost", "GPUCost", "DiskCost"})

		for _, cluster := range r.clusterData {

			allocationResult, err := r.allocations(ctx, cluster.name, date)
			if err != nil {
				log.Fatalf("Error: %v", err)
			}

			// allocation.name is the namespace name
			for _, data := range allocationResult.Data {
				for _, allocation := range data {
					log.Printf("Writing results for namespace: %v\n", allocation.Name)
					csv.Write([]string{
						cluster.name,
						cluster.subscription,
						util.GetOrDefault(cluster.namespaces[allocation.Name].Labels, "finance.ssc-spc.gc.ca/workload-id", ""),
						util.GetOrDefault(cluster.namespaces[allocation.Name].Annotations, "owner", ""),
						allocation.Name,
						allocation.Start.Format("2006-01-02"),
						strconv.FormatFloat(float64(allocation.CPUCost), 'f', -1, 32),
						strconv.FormatFloat(float64(allocation.RAMCost), 'f', -1, 32),
						strconv.FormatFloat(float64(allocation.GPUCost), 'f', -1, 32),
						strconv.FormatFloat(float64(allocation.PVCost), 'f', -1, 32),
					})
				}
			}
		}
		csv.Flush()
		if err := csv.Error(); err != nil {
			log.Fatalf("Error writing to CSV writer (report): %v", err)
		}
		pipeWriter.Close()
	}()

	return pipeReader, nil
}

// Extract costs from Kubecost
func (r *ReportGenerator) allocations(ctx context.Context, clusterName string, date time.Time) (*kubecost.AllocationResponse, error) {

	allocation, err := r.clusterData[clusterName].kubecostClient.Allocation(
		ctx,
		kubecost.DateWindow(date, date.AddDate(0, 0, 1)),
		kubecost.Aggregate([]string{"namespace"}),

		// We only want actual costs
		kubecost.Accumulate(true),
		kubecost.Idle(false),
		kubecost.External(false),
		kubecost.ShareIdle(false),
		kubecost.ShareTenancyCosts(false),
	)
	if err != nil {
		return nil, fmt.Errorf("Error getting Kubecost allocation for %s: %w", date, err)
	}

	return allocation, nil
}
