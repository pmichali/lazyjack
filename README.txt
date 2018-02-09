

FUTURE:
- Setup /etc/hosts and /etc/resolv.conf
- Prepare/Clean: NAT64, CNI plugin
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
