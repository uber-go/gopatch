Take the following steps to release a new version of gopatch.

1. Ensure that the CHANGELOG.md has an entry for every **user-facing** change.
   Do not add entries for changes that are not user-facing.

2. Change the "Unreleased" header to the target version number *without* the
   `v` prefix and add today's date in YYYY-MM-DD format. For example, if the
   target version is `v1.2.3`, add:

    ```diff
    -## Unreleased
    +## 1.2.3 - 2021-08-18
    ```

3. Create a new PR with the change and the following title:

    ```
    Preparing release v1.2.3
    ```

4. After landing the PR, tag the release with an **annotated** git tag and push
   the tag.

    ```
    $ git pull
    $ git tag -a v1.2.3 -m v1.2.3
    $ git push origin v1.2.3
    ```
