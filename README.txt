

TODOS/FUTURES (in no particular order):
- Decide how to handle failures on prepare. Exit? Rollback or rely on clean?
- Setup /etc/hosts and /etc/resolv.conf.
- See if way to use netlink to create/delete routes easily (without going through contortions to modify an existing route).
- Could do "ip route" and then regexp for route to delete, as a qualifier on route delete operation (or check for error=2 and ignore error).
- Prepare/Clean: CNI plugin
- Up & Down implementation.
- Packaging and pushing coe up to github.com.
- Validation of docker version. Kubernetes? Any other tools required?
- If needed/desired, see if can run DNS64 and NAT64 on separate hosts. Are routes correct?
- Check IP addresses, subnets, CIDRs in config to see if they are valid.
- Check that NAT IP within NAT subnet, and that NAT subnet withing support subnet.
- Mocking of netlink library and docker for better code coverage.
- Try this with IPv4 settings? Useful?
- Figure out how to do join on minions - need token.
- Support Calico plugin vs Bridge. Cillium? Others?
- Do Istio startup. Useful?  Metal LB startup?
- Should we support hypervisors other than Docker (have separated out the code)?
- Add function documentation.
- Check coverage, gofmt.
- Vendoring? of dependencies?
- Add version command
- Refactor
- Blog: prerequisites, dependencies (tayga/bind9, go packages, versions) how to create config file, usage, TODOs/Futures.
