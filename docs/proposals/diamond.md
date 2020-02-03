# Diamond workflow support.

Diamond workflow allows for a pipeline to scale out the generation of a single bundle. 

A diamond is composed of multiple splits; each split tries to write a set of files. Ideally, 2 splits do not write the 
same file.

To elaborate, a bundle consists of a filesystem namespace with files in folders; each split can write files to any 
folder but should not try to write the same file.

The individual nodes add files to splits without any co-ordination between them.
Commit logic handles any conflicts that exist. Conflicts can only occur if the same file in the namespace of the bundle 
is written to by more than one split.

The diamond workflow consists of the following steps.
1. Before a diamond starts, Datamon SDK allows for the creation of a unique ID for a diamond.
   2. The client is expected to use the id at each step.
   3. Command where client assigns an id: 
   ```datamon bundle split initialize --repo {repo} --message {message} --id {client assigned id}```
      1. The id can only be used once. 
   4. Command where datamon assigns an id.
      ```datamon bundle split initialize --repo {repo} --message {message}```
         1. Each invocation results in a new ID. 
   4. backend model: the following path is created in the vmetadata bucket.
      ```/splits/{repo}/{id}/split.yaml```   
1. Each split in a diamond generates a file list in the versioned metadata bucket.
   3. command: 
   ```datamon bundle split add --repo {repo} --id {ID} --source {path}```
   4. backend model: In the vmetadata bucket the following path will be created or rewritten.
      ```/splits/{repo}/{id}/filelist-{ksuid}.yaml```. KSUID suffix generated at the start of the add command allows 
      for the ordering of uploads.
   5. Incremental uploads for a bundle with or without splits work the same way.
   6. The ordering of ```add``` can be strongly ordered based on a design similar to WAL. (This is a nice to have 
   addition and can be added later in the implementation.)
1. In the end, the file lists combine into a bundle with the commit message.
    1. The bundle has a single namespace, and if 2 nodes in the diamond have duplicate file names with different data, 
    one of the conflicting writes wins. 
    The other files including the winner will be stored in a special path in the bundle. 
    (```./.{ID}/conflicts/{orig path}/file-suffix of client ID```)
    2. command: 
    ```datamon bundle split commit --repo {repo} --id {id} --message {message} --label {label}```
    3. Optionally a commit can be set to fail on conflict.
        1. command: 
        ```datamon bundle diamond commit --repo {repo} --id {id} --message {message} --label {label} --no-confict true``` 
    4. A file is written to mark the completion of a diamond and the corresponding bundle ID is written. 
    On failure 2 bundles can be created in the repo.
    for the same diamond but only one will win closing the bundle. ```/splits/{repo}/{id}/bundle.yaml```
## Other commands
### List all the splits in a repo
```datamon bundle split list --repo {repo}```
### List files in a split
```datamon bundle split list-files --repo {repo} --id {id}```

