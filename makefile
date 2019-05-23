start:
	-docker-compose down
	docker-compose up -d --force-recreate --remove-orphans
	# vault custom hackery
	# -docker-compose exec vault vault secrets disable secret
	-docker-compose exec vault vault secrets enable consul
	-docker-compose exec vault vault write consul/config/access \
		address=consul:8500 \
    	token=d9f1928e-1f84-407c-ab50-9579de563df5
	-docker-compose exec vault vault write consul/roles/myrole policy=$$(base64 <<< 'key "" { policy = "write" }')
	-docker-compose exec vault vault kv put secret/my-secret my-value=thisisversion1
	-docker-compose exec vault vault kv put secret/my-secret my-value=thisisversion2
	-docker-compose exec vault vault secrets enable -version=1 -path=secretv1 kv
	-docker-compose exec vault vault write secretv1/my-secret my-value1=versionlesssecret
	go run main.go
