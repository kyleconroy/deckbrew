## Pagination

Requests that return multiple items will be paginated to 100 items by default.
You can specify further pages with the ?page parameter.

    $ curl https://api.deckbrew.com/mtg/cards?page=2

Note that page numbering is 1-based and that omitting the ?page parameter will
return the first page.

### Link Header

The pagination info is included in the Link header. It is important to follow
these Link header values instead of constructing your own URLs. In some
instances, such as in the Commits API, pagination is based on SHA1 and not on
page number.

    Link: <https://api.deckbrew.com/mtg/cards?page=3>; rel="next",
      <https://api.deckbrew.com/mtg/cards?page=1>; rel="prev"

Linebreak is included for readability.

The possible rel values are:

Name 	| Description
next 	| Shows the URL of the immediate next page of results.
prev 	| Shows the URL of the immediate previous page of results.

## Query Language

The query language for the API is a subset of the [Elastic Search Query String Syntax].

The query string “mini-language” is used by the Query String Query and Field
Query, by the q query string parameter in the search API and by the percolate
parameter in the index and bulk APIs.

The query string is parsed into a series of terms and operators. A term can be
a single word — quick or brown — or a phrase, surrounded by double
quotes — "quick brown" — which searches for all the words in the phrase, in the
same order.

Operators allow you to customize the search — the available options are explained below.
Field names

As mentioned in Query String Query, the default_field is searched for the
search terms, but it is possible to specify other fields in the query syntax:
where the status field contains active

    status:active

where the title field contains quick or brown

    title:(quick brown)

where the author field contains the exact phrase "john smith"

    author:"John Smith"

### Ranges

**Probably just the simple syntax at the end**

Ranges can be specified for date, numeric or string fields. Inclusive ranges
are specified with square brackets [min TO max] and exclusive ranges with curly
brackets {min TO max}.

All days in 2012:

    date:[2012/01/01 TO 2012/12/31]

Numbers 1..5

    count:[1 TO 5]

Tags between alpha and omega, excluding alpha and omega:

    tag:{alpha TO omega}

Numbers from 10 upwards

    count:[10 TO *]

Dates before 2012

    date:{* TO 2012/01/01}

Curly and square brackets can be combined:

    Numbers from 1 up to but not including 5

    count:[1..5]

Ranges with one side unbounded can use the following syntax:

    age:>10
    age:>=10
    age:<10
    age:<=10

Note

To combine an upper and lower bound with the simplified syntax, you would need
to join two clauses with an AND operator:

    age:(+>=10 +<20)

The parsing of ranges in query strings can be complex and error prone. It is
much more reliable to use an explicit range filter.

### Boolean operators

By default, all terms are optional, as long as one term matches. A search for
foo bar baz will find any document that contains one or more of foo or bar or
baz. We have already discussed the default_operator above which allows you to
force all terms to be required, but there are also boolean operators which can
be used in the query string itself to provide more control.

The preferred operators are + (this term must be present) and - (this term must
not be present). All other terms are optional. For example, this query:

    quick brown +fox -news

states that:

    fox must be present
    news must not be present
    quick and brown are optional — their presence increases the relevance 

The familiar operators AND, OR and NOT (also written &&, || and !) are also
supported. However, the effects of these operators can be more complicated than
is obvious at first glance. NOT takes precedence over AND, which takes
precedence over OR. While the + and - only affect the term to the right of the
operator, AND and OR can affect the terms to the left and right.

### Grouping

Multiple terms or clauses can be grouped together with parentheses, to form
sub-queries:

(quick OR brown) AND fox

Groups can be used to target a particular field, or to boost the result of a
sub-query:

status:(active OR pending) title:(full text search)^2

Reserved characters

If you need to use any of the characters which function as operators in your
query itself (and not as operators), then you should escape them with a leading
backslash. For instance, to search for (1+1)=2, you would need to write your
query as \(1\+1\)=2.

The reserved characters are: + - && || ! ( ) { } [ ] ^ " ~ * ? : \ /

Failing to escape these special characters correctly could lead to a syntax error which prevents your query from running.

Watch this space

A space may also be a reserved character. For instance, if you have a synonym
list which converts "wi fi" to "wifi", a query_string search for "wi fi" would
fail. The query string parser would interpret your query as a search for "wi OR
fi", while the token stored in your index is actually "wifi". Escaping the
space will protect it from being touched by the query string parser: "wi\ fi".
Empty Query

If the query string is empty or only contains whitespaces the query string is
interpreted as a no_docs_query and will yield an empty result set.


[stash]: http://www.elasticsearch.org/guide/en/elasticsearch/reference/current/query-dsl-query-string-query.html#query-string-syntax
