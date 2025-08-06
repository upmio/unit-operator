package cmd

// Mock os.Exit for testing
var exitCode int

func mockExit(code int) {
	exitCode = code
}

// Test function for Execute
//func TestExecute(t *testing.T) {
//	// Table-driven test cases
//	tests := []struct {
//		name         string
//		setup        func()
//		expectedExit int
//	}{
//		{
//			name: "Successful Execute",
//			setup: func() {
//				rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
//					return nil
//				}
//			},
//			expectedExit: 0,
//		},
//		{
//			name: "Execute with error",
//			setup: func() {
//				rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
//					return errors.New("test error")
//				}
//			},
//			expectedExit: 1,
//		},
//	}
//
//	// Mock os.Exit
//	//oldExit := os.Exit
//	//os.Exit = mockExit
//	//defer func() { osExit = oldExit }()
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			// Reset exit code
//			exitCode = 0
//
//			// Setup the test case
//			tt.setup()
//
//			// Execute the function
//			Execute()
//
//			// Verify the result
//			if exitCode != tt.expectedExit {
//				t.Errorf("expected exit code %d, but got %d", tt.expectedExit, exitCode)
//			}
//		})
//	}
//}
