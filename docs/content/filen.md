---
title: "Filen"
description: "Rclone docs for Filen"
versionIntroduced: "TODO"
status: Alpha
---

# {{< icon "fa fa-solid fa-f" >}} Filen
## Configuration
The initial setup for Filen requires that you get an API key for your account,
currently this is only possible using the [Filen CLI](https://github.com/FilenCloudDienste/filen-cli).
This means you must first download the CLI, login, and then run the `export-api-key` command.

Here is an example of how to make a remote called `FilenRemote`.  First run:

     rclone config

This will guide you through an interactive setup process:
```
No remotes found, make a new one?
n) New remote
s) Set configuration password
q) Quit config
n/s/q> n

name> FilenRemote
Option Storage.

Type of storage to configure.
Choose a number from below, or type in your own value.
[snip]
XX / Filen
   \ "filen"
[snip]
Storage> filen

Option Email.
The email of your Filen account
Enter a value.
Email> youremail@provider.com

Option Password.
The password of your Filen account
Choose an alternative below.
y) Yes, type in my own password
g) Generate random password
y/g> y
Enter the password:
password:
Confirm the password:
password:

Option API Key.
An API Key for your Filen account
Get this using the Filen CLI export-api-key command
You can download the Filen CLI from https://github.com/FilenCloudDienste/filen-cli
Choose an alternative below.
y) Yes, type in my own password
g) Generate random password
y/g> y
Enter the password:
password:
Confirm the password:
password:

Edit advanced config?
y) Yes
n) No (default)
y/n> n

Configuration complete.
Options:
- type: filen
- Email: youremail@provider.com
- Password: *** ENCRYPTED ***
- API Key: *** ENCRYPTED ***
Keep this "FilenRemote" remote?
y) Yes this is OK (default)
e) Edit this remote
d) Delete this remote
y/e/d> y
```

### Modification times and hashes
Modification times are fully supported for files, for directories, only the creation time matters.

Filen supports SHA512 hashes.

### Restricted filename characters
Invalid UTF-8 bytes will be [replaced](/overview/#invalid-utf8)


### API Key

{{< rem autogenerated options start" - DO NOT EDIT - instead edit fs.RegInfo in backend/filen/filen.go then run make backenddocs" >}}
=== filen-rclone ===
### Standard options

Here are the Standard options specific to filen (Filen).

#### --filen-email

Email of your Filen account

Properties:

- Config:      email
- Env Var:     RCLONE_FILEN_EMAIL
- Type:        string
- Required:    true

#### --filen-password

Password of your Filen account

**NB** Input to this must be obscured - see [rclone obscure](/commands/rclone_obscure/).

Properties:

- Config:      password
- Env Var:     RCLONE_FILEN_PASSWORD
- Type:        string
- Required:    true

#### --filen-api-key

API Key for your Filen account 

Get this using the Filen CLI export-api-key command
You can download the Filen CLI from https://github.com/FilenCloudDienste/filen-cli

**NB** Input to this must be obscured - see [rclone obscure](/commands/rclone_obscure/).

Properties:

- Config:      api_key
- Env Var:     RCLONE_FILEN_API_KEY
- Type:        string
- Required:    true

### Advanced options

Here are the Advanced options specific to filen (Filen).

#### --filen-encoding

The encoding for the backend.

See the [encoding section in the overview](/overview/#encoding) for more info.

Properties:

- Config:      encoding
- Env Var:     RCLONE_FILEN_ENCODING
- Type:        Encoding
- Default:     Slash,Del,Ctl,InvalidUtf8,Dot

#### --filen-max-download-threads

Max number of threads to use when downloading files.

Each thread uses up to a bit over 1 MiB of memory.


Properties:

- Config:      max_download_threads
- Env Var:     RCLONE_FILEN_MAX_DOWNLOAD_THREADS
- Type:        int
- Default:     32

#### --filen-max-download-threads-per-file

Max number of threads per file to use when downloading files.

Each thread uses up to a bit over 1 MiB of memory.


Properties:

- Config:      max_download_threads_per_file
- Env Var:     RCLONE_FILEN_MAX_DOWNLOAD_THREADS_PER_FILE
- Type:        int
- Default:     8

#### --filen-max-upload-threads

Max number of threads to use when uploading files.

Each thread uses up to a bit over 1 MiB of memory.


Properties:

- Config:      max_upload_threads
- Env Var:     RCLONE_FILEN_MAX_UPLOAD_THREADS
- Type:        int
- Default:     64

#### --filen-master-keys

Master Keys (internal use only)

Properties:

- Config:      master_keys
- Env Var:     RCLONE_FILEN_MASTER_KEYS
- Type:        string
- Required:    false

#### --filen-private-key

Private RSA Key (internal use only)

Properties:

- Config:      private_key
- Env Var:     RCLONE_FILEN_PRIVATE_KEY
- Type:        string
- Required:    false

#### --filen-public-key

Public RSA Key (internal use only)

Properties:

- Config:      public_key
- Env Var:     RCLONE_FILEN_PUBLIC_KEY
- Type:        string
- Required:    false

#### --filen-auth-version

Authentication Version (internal use only)

Properties:

- Config:      auth_version
- Env Var:     RCLONE_FILEN_AUTH_VERSION
- Type:        string
- Required:    false

#### --filen-base-folder-uuid

UUID of Account Root Directory (internal use only)

Properties:

- Config:      base_folder_uuid
- Env Var:     RCLONE_FILEN_BASE_FOLDER_UUID
- Type:        string
- Required:    false

#### --filen-description

Description of the remote.

Properties:

- Config:      description
- Env Var:     RCLONE_FILEN_DESCRIPTION
- Type:        string
- Required:    false

=== filen-rclone ===
{{< rem autogenerated options stop >}}
