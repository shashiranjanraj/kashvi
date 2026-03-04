import subprocess
import re
import sys
import os

def check_coverage():
    print("Running go test with coverage...")
    try:
        # Run tests and generate coverage profile
        subprocess.run(
            ["go", "test", "-coverprofile=coverage.out", "./..."],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Get coverage function output
        result = subprocess.run(
            ["go", "tool", "cover", "-func=coverage.out"],
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        # Clean up coverage.out file
        if os.path.exists("coverage.out"):
            os.remove("coverage.out")
            
        # Parse the total coverage from the last line (e.g., "total:      (statements)    85.0%")
        lines = result.stdout.strip().split('\n')
        last_line = lines[-1]
        
        if "total:" in last_line:
            match = re.search(r'(\d+\.\d+)%', last_line)
            if match:
                percentage = float(match.group(1))
                print(f"Total Test Coverage: {percentage}%")
                return percentage
                
        print("Could not parse total coverage.")
        print("Output:", last_line)
        return 0.0
        
    except subprocess.CalledProcessError as e:
        print(f"Error running tests.")
        print(f"STDOUT: {e.stdout}")
        print(f"STDERR: {e.stderr}")
        sys.exit(1)
        
    except FileNotFoundError:
        print("Error: 'go' command not found. Ensure Go is installed and in your PATH.")
        sys.exit(1)

if __name__ == "__main__":
    check_coverage()
