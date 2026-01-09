package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/n0roo/pal-kit/internal/db"
	"github.com/n0roo/pal-kit/internal/lock"
	"github.com/spf13/cobra"
)

var (
	lockSessionID string
	lockForce     bool
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock 관리",
	Long:  `리소스 Lock을 관리합니다.`,
}

var lockAcquireCmd = &cobra.Command{
	Use:   "acquire <resource>",
	Short: "Lock 획득",
	Args:  cobra.ExactArgs(1),
	RunE:  runLockAcquire,
}

var lockReleaseCmd = &cobra.Command{
	Use:   "release <resource>",
	Short: "Lock 해제",
	Args:  cobra.ExactArgs(1),
	RunE:  runLockRelease,
}

var lockListCmd = &cobra.Command{
	Use:   "list",
	Short: "활성 Lock 목록",
	RunE:  runLockList,
}

var lockClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "모든 Lock 정리",
	RunE:  runLockClear,
}

func init() {
	rootCmd.AddCommand(lockCmd)
	lockCmd.AddCommand(lockAcquireCmd)
	lockCmd.AddCommand(lockReleaseCmd)
	lockCmd.AddCommand(lockListCmd)
	lockCmd.AddCommand(lockClearCmd)

	lockAcquireCmd.Flags().StringVar(&lockSessionID, "session", "", "세션 ID")
	lockClearCmd.Flags().BoolVar(&lockForce, "force", false, "강제 실행")
}

func getLockService() (*lock.Service, func(), error) {
	database, err := db.Open(GetDBPath())
	if err != nil {
		return nil, nil, err
	}
	return lock.NewService(database), func() { database.Close() }, nil
}

func runLockAcquire(cmd *cobra.Command, args []string) error {
	resource := args[0]
	sessionID := lockSessionID
	if sessionID == "" {
		sessionID = os.Getenv("CLAUDE_SESSION_ID")
	}
	if sessionID == "" {
		sessionID = "unknown"
	}

	svc, cleanup, err := getLockService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Acquire(resource, sessionID); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2) // Hook에서 차단으로 처리
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":   "acquired",
			"resource": resource,
			"session":  sessionID,
		})
	} else {
		fmt.Printf("✓ Lock 획득: %s (session: %s)\n", resource, sessionID)
	}

	return nil
}

func runLockRelease(cmd *cobra.Command, args []string) error {
	resource := args[0]

	svc, cleanup, err := getLockService()
	if err != nil {
		return err
	}
	defer cleanup()

	if err := svc.Release(resource); err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]string{
			"status":   "released",
			"resource": resource,
		})
	} else {
		fmt.Printf("✓ Lock 해제: %s\n", resource)
	}

	return nil
}

func runLockList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := getLockService()
	if err != nil {
		return err
	}
	defer cleanup()

	locks, err := svc.List()
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"locks": locks,
		})
		return nil
	}

	if len(locks) == 0 {
		fmt.Println("활성 Lock이 없습니다.")
		return nil
	}

	fmt.Printf("%-20s %-20s %s\n", "RESOURCE", "SESSION", "ACQUIRED")
	fmt.Println("------------------------------------------------------------")
	for _, l := range locks {
		fmt.Printf("%-20s %-20s %s\n", l.Resource, l.SessionID, l.AcquiredAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runLockClear(cmd *cobra.Command, args []string) error {
	if !lockForce {
		return fmt.Errorf("--force 플래그가 필요합니다")
	}

	svc, cleanup, err := getLockService()
	if err != nil {
		return err
	}
	defer cleanup()

	count, err := svc.Clear()
	if err != nil {
		return err
	}

	if jsonOut {
		json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
			"status":  "cleared",
			"removed": count,
		})
	} else {
		fmt.Printf("✓ %d개의 Lock 정리됨\n", count)
	}

	return nil
}
