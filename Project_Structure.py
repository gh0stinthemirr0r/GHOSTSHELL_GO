import os
import datetime

# Define the start folder and output file
start_folder = "."
output_file = "./DirectoryStructure.txt"

# Define ignore patterns
ignore_patterns = [
    'midjourney',        # Ignore anything related to "midjourney"
    'rust',              # Ignore anything related to "rust"
    'grafana',           # Ignore anything related to "grafana"
    'node_modules',      # Ignore node_modules directory
    'venv',              # Ignore virtual environment directories
    '__pycache__',       # Ignore Python cache folders
    '.git',              # Ignore Git folder
    '.DS_Store',         # Ignore macOS system files
]

# Define file extensions to ignore
ignore_file_extensions = [
    '.png',              # Ignore PNG files (images)
    '.svg',              # Ignore SVG files (vector images)
    '.log',              # Ignore log files
    '.zip',              # Ignore zip archives
    '.tar.gz',           # Ignore tar archives
    '.md',               # Ignore markdown files (e.g., README.md)
    '.lock',             # Ignore lock files (e.g., package-lock.json)
    '.pyc',              # Ignore Python compiled files
    '.iml',              # Ignore IntelliJ Idea project files
    '.class'             # Ignore Java compiled class files
]

def should_ignore(path):
    # Check if the path matches any of the ignore patterns
    for pattern in ignore_patterns:
        if pattern in path:
            return True
    # Check if the file matches any of the ignore extensions
    for ext in ignore_file_extensions:
        if path.endswith(ext):
            return True
    return False

def get_directory_structure(path, depth=0):
    try:
        # List all items in the directory
        for item in os.listdir(path):
            full_path = os.path.join(path, item)
            relative_path = os.path.relpath(full_path, start_folder)

            # If the item matches the ignore rules, skip it
            if should_ignore(full_path):
                continue

            # Get last modified date
            modified_time = datetime.datetime.fromtimestamp(os.path.getmtime(full_path)).strftime('%Y-%m-%d %H:%M:%S')

            # Print and add directory or file to the output
            indent = " " * (depth * 2)
            if os.path.isdir(full_path):
                # It's a directory
                line = f"{indent}[📁] Folder: {item} ({relative_path}) - Last Modified: {modified_time}"
                add_to_output(line)
                # Recursively call for subdirectories
                get_directory_structure(full_path, depth + 1)
            else:
                # It's a file
                line = f"{indent}[📄] File: {item} in ({relative_path}) - Last Modified: {modified_time}"
                add_to_output(line)

    except PermissionError:
        # Skip directories for which we don't have permission
        pass

def add_to_output(line):
    with open(output_file, "a", encoding="utf-8") as file:
        file.write(line + "\n")

if __name__ == "__main__":
    # Create or overwrite the output file with the header
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    header = f"Project Directory Structure for: {os.path.abspath(start_folder)}\nCaptured on: {timestamp}\n{'-'*40}\n"
    with open(output_file, "w", encoding="utf-8") as file:
        file.write(header)

    # Get the directory structure
    get_directory_structure(start_folder)

    # Inform user that the directory structure has been printed
    print(f"Directory structure has been written to {output_file}")
