# One-time bootstrap: installs the kashvi CLI binary.
# After this, use kashvi commands directly:
#
#   kashvi run              # start the server
#   kashvi build            # compile ./kashvi binary
#   kashvi migrate          # run migrations
#   kashvi migrate:rollback # rollback last batch
#   kashvi migrate:status   # show migration status
#   kashvi seed             # run all seeders
#   kashvi queue:work       # start queue worker
#   kashvi schedule:run     # start task scheduler
#   kashvi make:model Name
#   kashvi make:controller Name
#   kashvi make:service Name
#   kashvi make:migration name
#   kashvi make:seeder Name

.PHONY: install
install:
	go install ./cmd/kashvi/