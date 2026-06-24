Contributing to cerebro
=============================

You can contribute to cerebro through code, documentation or bug reports.

## Bug reports

For bug reports, make sure you:
- give enough details regarding your cerebro setup
- include versions for elasticsearch and cerebro
- describe steps to reproduce the error, the error itself and expected behaviour

## Pull requests

Before getting started on a pull request(be it a new feature or a bug fix), please open an issue explaning what you would like to achieve and how you would go about this.
Even though I'm open to feature requests, I might not always agree on the value a feature might bring. And I would hate to waste someone else's time.

Once working on a pull request, please:
- include the generated frontend files (`npm run build`)
- add tests that validate your changes
- squash your development commits to keep only important commits(fix typo, wrong indent should not be part of git history)
- rebase it against development before submiting
- make sure all tests pass (`npm test`)

## Development

Generate the API client and run the frontend build:

```sh
npm run api:generate
npm run build
```

Run the Go server and, in another terminal, Vite for frontend development:

```sh
go run ./cmd/cerebro serve -config conf/application.dev.yaml
npm run dev
```
