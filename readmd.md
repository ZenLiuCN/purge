# purge

purge is a simple ugly command line tool for clean temporary files in file tree.

## action

1. read `.gitingore` for files should be cleaned.
2. read `.keep` for files should be kept, this has the highest priority.
3. read `purge.conf` in the same directory of executable for global configurations.

## simple usage

`purge`  print help info.

`purge -h`  print help info.

`purge -l`  running under current working directory and logging purge list into log file.

`purge -p`  execute purge and print in console.

## flags and parameters

1. `-l` instead print in console, save logging into file.
2. `-t` test mode, print details includes configuration and matches.
3. `-p` execute physical purge, this won't work with `-t`.
4. `-r=true` print ignored files.
5. `-d=path` use supplied path instead of current working directory.
6. `-h` print simple help.
7. `-i=pattern -i=pattern` temporary keep patterns
8. `-f=file-name` temporary loading ignore files

## config

1. format using `HOCON`,must put at the same location of executable and with name `purge.conf`.
2. `files`: list of file names that contain patterns, `.gitignore` is always included.
3. `gabage`: list of pattern that should purge, there have no default values.
4. `keep`: list of pattern that should never purge, `.git/` is always included.
5. see `purge.conf` in source as an example.

## .keep file

use same format of `.gitignore`, but file matched any pattern in `.keep` won't been purged.