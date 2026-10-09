package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	accountDomain "github.com/sekai-labs/kumokura/internal/accounts/domain"
	accountPorts "github.com/sekai-labs/kumokura/internal/accounts/ports"
	"github.com/sekai-labs/kumokura/internal/bootstrap"
	bucketDomain "github.com/sekai-labs/kumokura/internal/buckets/domain"
	objectDomain "github.com/sekai-labs/kumokura/internal/objects/domain"
	"github.com/sekai-labs/kumokura/internal/platform/config"
	"github.com/sekai-labs/kumokura/internal/presentation/tui"
	syncDomain "github.com/sekai-labs/kumokura/internal/synchronization/domain"
	syncPorts "github.com/sekai-labs/kumokura/internal/synchronization/ports"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	JSONOutput bool
	Account    string
}

func NewRootCmd(app *bootstrap.AppContainer, out, err io.Writer) *cobra.Command {
	opts := &RootOptions{}

	rootCmd := &cobra.Command{
		Use:           "kumokura",
		Short:         "Kumokura - high-performance cross-cloud S3 storage manager",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.SetOut(out)
	rootCmd.SetErr(err)

	rootCmd.PersistentFlags().BoolVar(&opts.JSONOutput, "json", false, "Output data in JSON format")
	rootCmd.PersistentFlags().StringVar(&opts.Account, "account", "", "Target storage account name")

	rootCmd.AddCommand(newVersionCmd(opts, out, err))
	rootCmd.AddCommand(newAccountCmd(app, opts, out, err))
	rootCmd.AddCommand(newBucketCmd(app, opts, out, err))
	rootCmd.AddCommand(newObjectCmd(app, opts, out, err))
	rootCmd.AddCommand(newTransferCmd(app, opts, out, err))
	rootCmd.AddCommand(newSyncCmd(app, opts, out, err))
	rootCmd.AddCommand(newConfigCmd(app, opts, out, err))
	rootCmd.AddCommand(newTUICmd(app, opts, out, err))

	return rootCmd
}

func newVersionCmd(opts *RootOptions, out, err io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Kumokura version",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{
					"version": "0.1.1",
					"commit":  "clean",
					"build":   "go1.24",
				})
			}
			f.PrintMessage("kumokura v0.1.1")
			return nil
		},
	}
}

func newAccountCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage storage accounts and credentials",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			accounts, err := app.AccountService.ListAccounts(cmd.Context())
			if err != nil {
				f.PrintError("failed to list accounts: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(accounts)
			}
			headers := []string{"ID", "NAME", "TYPE", "REGION", "ENDPOINT"}
			var rows [][]string
			for _, a := range accounts {
				rows = append(rows, []string{
					string(a.ID),
					a.Name,
					string(a.Type),
					a.Region,
					a.Endpoint,
				})
			}
			return f.PrintTable(headers, rows)
		},
	}

	var (
		accName      string
		accType      string
		endpoint     string
		region       string
		pathStyle    bool
		accessKey    string
		secretKey    string
		sessionToken string
	)

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new storage account",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if accessKey == "" {
				accessKey = os.Getenv("KUMOKURA_ACCESS_KEY")
			}
			if secretKey == "" {
				secretKey = os.Getenv("KUMOKURA_SECRET_KEY")
			}
			if sessionToken == "" {
				sessionToken = os.Getenv("KUMOKURA_SESSION_TOKEN")
			}
			creds, err := accountDomain.NewCredentials(accessKey, secretKey, sessionToken)
			if err != nil {
				f.PrintError("invalid credentials: %v", err)
				return err
			}

			acc, err := app.AccountService.CreateAccount(cmd.Context(), accountPorts.CreateAccountParams{
				Name:         accName,
				Type:         accountDomain.AccountType(accType),
				Endpoint:     endpoint,
				Region:       region,
				UsePathStyle: pathStyle,
				Credentials:  creds,
			})
			if err != nil {
				f.PrintError("failed to create account: %v", err)
				return err
			}

			if opts.JSONOutput {
				return f.PrintJSON(acc)
			}
			f.PrintMessage("Account %s (%s) created successfully.", acc.Name, acc.ID)
			return nil
		},
	}
	addCmd.Flags().StringVar(&accName, "name", "", "Account name (required)")
	addCmd.Flags().StringVar(&accType, "type", "AWS", "Account type (AWS, MinIO, CloudflareR2, etc.)")
	addCmd.Flags().StringVar(&endpoint, "endpoint", "", "Custom S3 endpoint")
	addCmd.Flags().StringVar(&region, "region", "us-east-1", "Region")
	addCmd.Flags().BoolVar(&pathStyle, "path-style", false, "Use path-style addressing")
	addCmd.Flags().StringVar(&accessKey, "access-key", "", "Access Key ID (optional if KUMOKURA_ACCESS_KEY set)")
	addCmd.Flags().StringVar(&secretKey, "secret-key", "", "Secret Access Key (optional if KUMOKURA_SECRET_KEY set)")
	addCmd.Flags().StringVar(&sessionToken, "session-token", "", "Optional session token")
	_ = addCmd.MarkFlagRequired("name")

	removeCmd := &cobra.Command{
		Use:   "remove <account-id>",
		Short: "Remove a storage account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			id := accountDomain.AccountID(args[0])
			if err := app.AccountService.DeleteAccount(cmd.Context(), id); err != nil {
				f.PrintError("failed to delete account: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "deleted", "id": args[0]})
			}
			f.PrintMessage("Account %s removed.", args[0])
			return nil
		},
	}

	testCmd := &cobra.Command{
		Use:   "test <account-id>",
		Short: "Test connection to an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			id := accountDomain.AccountID(args[0])
			if err := app.AccountService.TestConnection(cmd.Context(), id); err != nil {
				f.PrintError("connection test failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "success", "id": args[0]})
			}
			f.PrintMessage("Connection to account %s verified successfully.", args[0])
			return nil
		},
	}

	defaultCmd := &cobra.Command{
		Use:   "default [account-name]",
		Short: "Get or set default account",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if len(args) == 0 {
				if opts.JSONOutput {
					return f.PrintJSON(map[string]string{"default_account": app.Config.LogLevel})
				}
				f.PrintMessage("Default account not configured.")
				return nil
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"default_account": args[0]})
			}
			f.PrintMessage("Default account set to %s.", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, addCmd, removeCmd, testCmd, defaultCmd)
	return cmd
}

func newBucketCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bucket",
		Short: "Manage S3 buckets",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all buckets",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			buckets, lErr := bSvc.ListBuckets(cmd.Context())
			if lErr != nil {
				f.PrintError("list buckets failed: %v", lErr)
				return lErr
			}
			if opts.JSONOutput {
				return f.PrintJSON(buckets)
			}
			headers := []string{"NAME", "REGION", "CREATED_AT"}
			var rows [][]string
			for _, b := range buckets {
				rows = append(rows, []string{
					b.Name,
					b.Region,
					b.CreationDate.Format("2006-01-02 15:04:05"),
				})
			}
			return f.PrintTable(headers, rows)
		},
	}

	var bucketRegion string
	createCmd := &cobra.Command{
		Use:   "create <bucket-name>",
		Short: "Create a bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			if err := bSvc.CreateBucket(cmd.Context(), args[0], bucketRegion); err != nil {
				f.PrintError("create bucket failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "created", "bucket": args[0]})
			}
			f.PrintMessage("Bucket %s created.", args[0])
			return nil
		},
	}
	createCmd.Flags().StringVar(&bucketRegion, "region", "us-east-1", "Bucket region")

	deleteCmd := &cobra.Command{
		Use:   "delete <bucket-name>",
		Short: "Delete a bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			if err := bSvc.DeleteBucket(cmd.Context(), args[0]); err != nil {
				f.PrintError("delete bucket failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "deleted", "bucket": args[0]})
			}
			f.PrintMessage("Bucket %s deleted.", args[0])
			return nil
		},
	}

	infoCmd := &cobra.Command{
		Use:   "info <bucket-name>",
		Short: "Get bucket information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			loc, err := bSvc.GetBucketLocation(cmd.Context(), args[0])
			if err != nil {
				f.PrintError("get bucket info failed: %v", err)
				return err
			}
			info := map[string]string{"bucket": args[0], "location": loc}
			if opts.JSONOutput {
				return f.PrintJSON(info)
			}
			return f.PrintTable([]string{"PROPERTY", "VALUE"}, [][]string{
				{"Bucket", args[0]},
				{"Region", loc},
			})
		},
	}

	taggingCmd := &cobra.Command{
		Use:   "tagging <bucket-name>",
		Short: "View bucket tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			tags, err := bSvc.GetBucketTags(cmd.Context(), args[0])
			if err != nil {
				f.PrintError("get tags failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(tags)
			}
			var rows [][]string
			for _, tag := range tags {
				rows = append(rows, []string{tag.Key, tag.Value})
			}
			return f.PrintTable([]string{"KEY", "VALUE"}, rows)
		},
	}

	versioningCmd := &cobra.Command{
		Use:   "versioning <bucket-name> [enable|suspend]",
		Short: "Get or set bucket versioning",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			bSvc, bErr := app.CreateBucketService(cmd.Context(), opts.Account)
			if bErr != nil {
				f.PrintError("initialize bucket service: %v", bErr)
				return bErr
			}
			if len(args) == 2 {
				status := bucketDomain.VersioningStatusSuspended
				if args[1] == "enable" {
					status = bucketDomain.VersioningStatusEnabled
				}
				if err := bSvc.SetBucketVersioning(cmd.Context(), args[0], bucketDomain.VersioningConfig{Status: status}); err != nil {
					f.PrintError("set versioning failed: %v", err)
					return err
				}
				if opts.JSONOutput {
					return f.PrintJSON(map[string]string{"bucket": args[0], "status": string(status)})
				}
				f.PrintMessage("Bucket %s versioning updated to %s.", args[0], status)
				return nil
			}

			status, err := bSvc.GetBucketVersioning(cmd.Context(), args[0])
			if err != nil {
				f.PrintError("get versioning failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"bucket": args[0], "versioning": string(status.Status)})
			}
			f.PrintMessage("Bucket %s versioning: %s", args[0], status.Status)
			return nil
		},
	}

	cmd.AddCommand(listCmd, createCmd, deleteCmd, infoCmd, taggingCmd, versioningCmd)
	return cmd
}

func newObjectCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "object",
		Short: "Manage S3 objects",
	}

	var prefix string
	var limit int32
	listCmd := &cobra.Command{
		Use:   "list <bucket>",
		Short: "List objects in bucket",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			res, err := objSvc.ListObjects(cmd.Context(), args[0], objectDomain.ObjectFilter{
				Prefix:  prefix,
				MaxKeys: limit,
			})
			if err != nil {
				f.PrintError("list objects failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(res.Objects)
			}
			headers := []string{"KEY", "SIZE", "STORAGE_CLASS", "MODIFIED"}
			var rows [][]string
			for _, obj := range res.Objects {
				rows = append(rows, []string{
					obj.Key,
					strconv.FormatInt(obj.Size, 10),
					string(obj.StorageClass),
					obj.LastModified.Format("2006-01-02 15:04:05"),
				})
			}
			return f.PrintTable(headers, rows)
		},
	}
	listCmd.Flags().StringVar(&prefix, "prefix", "", "Filter by prefix")
	listCmd.Flags().Int32Var(&limit, "limit", 1000, "Max objects to list")

	getCmd := &cobra.Command{
		Use:   "get <bucket> <key> <dest-file>",
		Short: "Download object",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			cleanDest := filepath.Clean(args[2])
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			content, err := objSvc.GetObject(cmd.Context(), args[0], args[1], "")
			if err != nil {
				f.PrintError("get object failed: %v", err)
				return err
			}
			defer content.Body.Close()

			outFile, err := os.OpenFile(cleanDest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				f.PrintError("create destination file failed: %v", err)
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, content.Body); err != nil {
				f.PrintError("write destination file failed: %v", err)
				return err
			}

			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "downloaded", "key": args[1], "dest": cleanDest})
			}
			f.PrintMessage("Object %s downloaded to %s.", args[1], cleanDest)
			return nil
		},
	}

	putCmd := &cobra.Command{
		Use:   "put <bucket> <key> <source-file>",
		Short: "Upload object",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			file, err := os.Open(args[2])
			if err != nil {
				f.PrintError("open source file failed: %v", err)
				return err
			}
			defer file.Close()
			stat, err := file.Stat()
			if err != nil {
				f.PrintError("stat source file failed: %v", err)
				return err
			}
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			_, err = objSvc.PutObject(cmd.Context(), args[0], args[1], file, stat.Size(), objectDomain.ObjectMetadata{})
			if err != nil {
				f.PrintError("put object failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "uploaded", "key": args[1], "bucket": args[0]})
			}
			f.PrintMessage("Object %s uploaded to %s.", args[1], args[0])
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete <bucket> <key>",
		Short: "Delete object",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			if err := objSvc.DeleteObject(cmd.Context(), args[0], args[1], ""); err != nil {
				f.PrintError("delete object failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "deleted", "key": args[1]})
			}
			f.PrintMessage("Object %s deleted from %s.", args[1], args[0])
			return nil
		},
	}

	copyCmd := &cobra.Command{
		Use:   "copy <src-bucket> <src-key> <dst-bucket> <dst-key>",
		Short: "Copy object",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			if err := objSvc.CopyObject(cmd.Context(), args[0], args[1], "", args[2], args[3]); err != nil {
				f.PrintError("copy object failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "copied", "src": args[1], "dst": args[3]})
			}
			f.PrintMessage("Object %s copied to %s.", args[1], args[3])
			return nil
		},
	}

	moveCmd := &cobra.Command{
		Use:   "move <src-bucket> <src-key> <dst-bucket> <dst-key>",
		Short: "Move object",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			if err := objSvc.MoveObject(cmd.Context(), args[0], args[1], args[2], args[3]); err != nil {
				f.PrintError("move object failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "moved", "src": args[1], "dst": args[3]})
			}
			f.PrintMessage("Object %s moved to %s.", args[1], args[3])
			return nil
		},
	}

	presignCmd := &cobra.Command{
		Use:   "presign <bucket> <key>",
		Short: "Generate presigned URL",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			pURL, err := objSvc.GeneratePresignedURL(cmd.Context(), args[0], args[1], "GET", 3600*time.Second)
			if err != nil {
				f.PrintError("presign failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"url": pURL.URL})
			}
			f.PrintMessage("%s", pURL.URL)
			return nil
		},
	}

	versionsCmd := &cobra.Command{
		Use:   "versions <bucket>",
		Short: "List object versions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			objSvc, oErr := app.CreateObjectService(cmd.Context(), opts.Account)
			if oErr != nil {
				f.PrintError("initialize object service: %v", oErr)
				return oErr
			}
			res, err := objSvc.ListObjectVersions(cmd.Context(), args[0], prefix, "", "", limit)
			if err != nil {
				f.PrintError("list versions failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(res.Versions)
			}
			headers := []string{"KEY", "VERSION_ID", "LATEST", "MODIFIED"}
			var rows [][]string
			for _, v := range res.Versions {
				rows = append(rows, []string{
					v.Key,
					v.VersionID,
					strconv.FormatBool(v.IsLatest),
					v.LastModified.Format("2006-01-02 15:04:05"),
				})
			}
			return f.PrintTable(headers, rows)
		},
	}

	cmd.AddCommand(listCmd, getCmd, putCmd, deleteCmd, copyCmd, moveCmd, presignCmd, versionsCmd)
	return cmd
}

func newTransferCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Manage file transfers",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List transfer jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			jobs, err := app.TransferRepo.ListJobs(cmd.Context(), opts.Account, "")
			if err != nil {
				f.PrintError("list transfers failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(jobs)
			}
			headers := []string{"ID", "STATUS", "SRC", "DST", "TRANSFERRED", "TOTAL"}
			var rows [][]string
			for _, j := range jobs {
				rows = append(rows, []string{
					j.ID,
					string(j.Status),
					j.SourcePath,
					j.DestinationPath,
					strconv.FormatInt(j.BytesTransferred, 10),
					strconv.FormatInt(j.TotalBytes, 10),
				})
			}
			return f.PrintTable(headers, rows)
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status <job-id>",
		Short: "Get transfer job status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			job, err := app.TransferRepo.GetJob(cmd.Context(), args[0])
			if err != nil {
				f.PrintError("get transfer job failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(job)
			}
			return f.PrintTable([]string{"PROPERTY", "VALUE"}, [][]string{
				{"ID", job.ID},
				{"Status", string(job.Status)},
				{"Source", job.SourcePath},
				{"Destination", job.DestinationPath},
				{"Transferred", strconv.FormatInt(job.BytesTransferred, 10)},
				{"TotalBytes", strconv.FormatInt(job.TotalBytes, 10)},
			})
		},
	}

	pauseCmd := &cobra.Command{
		Use:   "pause <job-id>",
		Short: "Pause transfer job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if err := app.TransferService.PauseJob(cmd.Context(), args[0]); err != nil {
				f.PrintError("pause transfer job failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "paused", "id": args[0]})
			}
			f.PrintMessage("Transfer job %s paused.", args[0])
			return nil
		},
	}

	resumeCmd := &cobra.Command{
		Use:   "resume <job-id>",
		Short: "Resume transfer job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if err := app.TransferService.ResumeJob(cmd.Context(), args[0]); err != nil {
				f.PrintError("resume transfer job failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "resumed", "id": args[0]})
			}
			f.PrintMessage("Transfer job %s resumed.", args[0])
			return nil
		},
	}

	cancelCmd := &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "Cancel transfer job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if err := app.TransferService.CancelJob(cmd.Context(), args[0]); err != nil {
				f.PrintError("cancel transfer job failed: %v", err)
				return err
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "cancelled", "id": args[0]})
			}
			f.PrintMessage("Transfer job %s cancelled.", args[0])
			return nil
		},
	}

	cmd.AddCommand(listCmd, statusCmd, pauseCmd, resumeCmd, cancelCmd)
	return cmd
}

func newSyncCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	var (
		dryRun      bool
		deleteEx    bool
		strategy    string
		conflict    string
		concurrency int
		includes    []string
		excludes    []string
	)
	cmd := &cobra.Command{
		Use:   "sync <source> <target>",
		Short: "Synchronize directories and S3 buckets",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			srcScanner, dstScanner, sErr := app.CreateSyncScanners(cmd.Context(), opts.Account, args[0], args[1])
			if sErr != nil {
				f.PrintError("initialize sync scanners: %v", sErr)
				return sErr
			}

			if concurrency <= 0 {
				concurrency = app.Config.MaxUploadConcurrency
				if concurrency <= 0 {
					concurrency = 100
				}
			}
			if fc, _ := config.LoadFolderConfig(args[0]); fc != nil && fc.MaxConcurrency > 0 {
				concurrency = fc.MaxConcurrency
			}

			syncOpts := syncPorts.SyncOptions{
				Direction:        syncDomain.DirectionLocalToS3,
				Mode:             syncDomain.ModeUploadOnly,
				Strategy:         syncDomain.ComparisonStrategy(strategy),
				ConflictPolicy:   syncDomain.ConflictPolicy(conflict),
				DeleteExtraneous: deleteEx,
				DryRun:           dryRun,
				MaxConcurrency:   concurrency,
				Filter: syncDomain.Filter{
					Includes: includes,
					Excludes: excludes,
				},
			}

			plan, pErr := app.SyncService.Plan(cmd.Context(), srcScanner, dstScanner, args[0], args[1], syncOpts)
			if pErr != nil {
				f.PrintError("generate sync plan failed: %v", pErr)
				return pErr
			}

			if dryRun || opts.JSONOutput {
				if opts.JSONOutput {
					return f.PrintJSON(plan)
				}
				headers := []string{"ACTION", "PATH", "SIZE", "REASON"}
				var rows [][]string
				for _, item := range plan.Items {
					rows = append(rows, []string{
						string(item.Action),
						item.RelativePath,
						strconv.FormatInt(item.SourceSize, 10),
						item.Reason,
					})
				}
				_ = f.PrintTable(headers, rows)
				f.PrintMessage("\nSummary: %d to upload, %d to delete, %d skips, %d conflicts (%d bytes total)",
					plan.TotalUploads, plan.TotalDeletes, plan.TotalSkips, plan.TotalConflicts, plan.TotalBytes)
				return nil
			}

			if err := app.SyncService.Execute(cmd.Context(), plan, syncOpts); err != nil {
				f.PrintError("sync execution failed: %v", err)
				return err
			}

			f.PrintMessage("Sync complete: %d uploaded, %d deleted.", plan.TotalUploads, plan.TotalDeletes)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview sync operations without executing")
	cmd.Flags().IntVar(&concurrency, "concurrency", 100, "Maximum concurrent upload/sync workers")
	cmd.Flags().BoolVar(&deleteEx, "delete", false, "Delete extraneous files in destination")
	cmd.Flags().StringVar(&conflict, "conflict", "KeepNewer", "Conflict resolution policy (Overwrite, Skip, KeepNewer, Fail)")
	cmd.Flags().StringSliceVar(&includes, "include", nil, "Glob patterns to include")
	cmd.Flags().StringSliceVar(&excludes, "exclude", nil, "Glob patterns to exclude")

	return cmd
}

func newConfigCmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Kumokura configuration",
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show active configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			cfg := app.Config
			if opts.JSONOutput {
				return f.PrintJSON(cfg)
			}
			return f.PrintTable([]string{"KEY", "VALUE"}, [][]string{
				{"ConfigDir", cfg.ConfigDir},
				{"DataDir", cfg.DataDir},
				{"DBPath", cfg.DBPath},
				{"SecretsDir", cfg.SecretsDir},
				{"LogLevel", cfg.LogLevel},
			})
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			val := ""
			switch args[0] {
			case "LogLevel":
				val = app.Config.LogLevel
			case "ConfigDir":
				val = app.Config.ConfigDir
			case "DataDir":
				val = app.Config.DataDir
			case "DBPath":
				val = app.Config.DBPath
			case "SecretsDir":
				val = app.Config.SecretsDir
			default:
				f.PrintError("unknown config key: %s", args[0])
				return fmt.Errorf("unknown key: %s", args[0])
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{args[0]: val})
			}
			f.PrintMessage("%s", val)
			return nil
		},
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := NewFormatter(out, err, opts.JSONOutput)
			if args[0] == "LogLevel" {
				app.Config.LogLevel = args[1]
			}
			if opts.JSONOutput {
				return f.PrintJSON(map[string]string{"status": "updated", args[0]: args[1]})
			}
			f.PrintMessage("Config %s updated to %s.", args[0], args[1])
			return nil
		},
	}

	cmd.AddCommand(showCmd, getCmd, setCmd)
	return cmd
}

func newTUICmd(app *bootstrap.AppContainer, opts *RootOptions, out, err io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive terminal user interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			services := tui.Services{
				AccountService:  app.AccountService,
				TransferService: app.TransferService,
				SyncService:     app.SyncService,
				SyncRepo:        app.SyncRepo,
			}

			targetAccount := opts.Account
			if targetAccount == "" {
				accounts, accErr := app.AccountService.ListAccounts(ctx)
				if accErr == nil && len(accounts) > 0 {
					targetAccount = accounts[0].Name
				}
			}

			if targetAccount != "" {
				bSvc, bErr := app.CreateBucketService(ctx, targetAccount)
				if bErr == nil {
					services.BucketService = bSvc
				}
				oSvc, oErr := app.CreateObjectService(ctx, targetAccount)
				if oErr == nil {
					services.ObjectService = oSvc
				}
			}

			model := tui.NewModel(services)
			p := tea.NewProgram(
				model,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)
			_, runErr := p.Run()
			return runErr
		},
	}
}
