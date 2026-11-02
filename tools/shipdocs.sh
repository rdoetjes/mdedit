#!/bin/sh
# Copies the documents that must go along with every copy of mdedit into a
# folder: the licences, and the EU declaration of conformity once signed
# (Cyber Resilience Act, Article 13(20)). A declaration still marked as a
# draft is left out: shipping it would publish an invalid declaration.
set -e
dest=$1
doc=docs/cra/declaration-of-conformity.md
cp LICENSE THIRD_PARTY_LICENSES.md "$dest"/
if grep -q '^> \*\*DRAFT' "$doc"; then
	echo "shipdocs: $doc is a draft, not included"
	rm -f "$dest/EU-declaration-of-conformity.md"
else
	cp "$doc" "$dest/EU-declaration-of-conformity.md"
fi
