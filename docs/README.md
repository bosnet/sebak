# API documents with [API blueprint](https://apiblueprint.org/)

- https://apiblueprint.org/documentation/tutorial.html

Install [aglio](https://github.com/danielgtaylor/aglio):

```
brew install node
npm config set prefix '~/.npm-global'
export PATH=~/.npm-global/bin:$PATH
npm install -g aglio
```

- https://docs.npmjs.com/getting-started/fixing-npm-permissions


Generate HTML document:

```
~/.npm-global/bin/aglio -i ./API.md -o api.html
```

Serve HTML document:

- https://bosnet.github.io/sebak/api/index.html

```
git add api.html
git commit -m 'add api.html'
git push origin gh-pages

```

- https://pages.github.com
- TODO: Intergate with CI to generate and serve html

---

> Current We don't use it. 

Install [snowboard](https://github.com/bukalapak/snowboard).

Generate HTML document:

```
$ snowboard html -o output API.md
```

Serve HTML document:

```
$ snowboard html -o output -s API.md
```

Open <http://localhost:8088>.

Validate API document:

```
$ snowboard lint API.md
```
