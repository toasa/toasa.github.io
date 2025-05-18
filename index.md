---
layout: default
title: アーカイブ
---
<h1>アーカイブ</h1>
<ul>
  {% assign years = "" | split: "" %}
  {% for p in site.pages %}
    {% if p.path contains '/' %}
      {% assign year = p.path | split: '/' | first %}
      {% comment %}年ディレクトリは4桁の数字と仮定 (例: "2023") {% endcomment %}
      {% if year != nil and year.size == 4 and year contains "20" %}
        {% unless years contains year %}
          {% assign years = years | push: year %}
        {% endunless %}
      {% endif %}
    {% endif %}
  {% endfor %}
  {% assign sorted_years = years | uniq | sort | reverse %}
  {% for year in sorted_years %}
    <li><a href="{{ site.baseurl }}/{{ year }}/">{{ year }}</a></li>
  {% endfor %}
</ul>
