import sys
import json
import copy

panelstart = 10

def process_first_row(row, environment):
    global panelstart
    panels = row['panels']
    for panel in panels:
        panel['id'] = panelstart
        panelstart += 1
        targets = panel['targets']
        for target in targets:
            target['expr'] = target['expr'].replace('{{{ENVIRONMENT}}}', environment)

def add_topics(jsond, topics, template, environment):
    global panelstart
    for topic in topics:
        tpl = copy.deepcopy(template)
        tpl['title'] = "Topic:" + topic
        panels = tpl['panels']
        panels[0]['id'] = panelstart
        panels[1]['id'] = panelstart + 1
        panelstart += 2
        panels[0]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{TOPIC}}}', topic)
        panels[0]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        panels[0]['title'] = panels[0]['title'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][0]['expr'] =\
            panels[1]['targets'][0]['expr'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][0]['expr'] =\
            panels[1]['targets'][0]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        panels[1]['targets'][1]['expr'] =\
            panels[1]['targets'][1]['expr'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][1]['expr'] =\
            panels[1]['targets'][1]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        jsond['rows'].append(tpl)


def add_consumer_group(jsond, group, topics, template, environment):
    global panelstart
    for topic in topics:
        tpl = copy.deepcopy(template)
        tpl['title'] = 'Group:{} Topic:{}'.format(group, topic)
        for panel in tpl['panels']:
            panel['id'] = panelstart
            panelstart += 1
            title = panel['title']
            title = title.replace('{{{CONSUMERGROUP}}}', group)
            title = title.replace('{{{CONSUMERTOPIC}}}', topic)
            panel['title'] = title
            for target in panel['targets']:
                expr = target['expr']
                expr = expr.replace('{{{ENVIRONMENT}}}', environment)
                expr = expr.replace('{{{CONSUMERGROUP}}}', group)
                expr = expr.replace('{{{CONSUMERTOPIC}}}', topic)
                target['expr'] = expr
        jsond['rows'].append(tpl)



def main():
    if len(sys.argv) < 2:
        print('Usage: {} <environment_name'.format(sys.argv[0]))
        exit(1)
    environname = sys.argv[1]
    jsondata = json.load(open('template.json', 'r'))
    jsoninput = json.load(open('input.json', 'r'))
    topictemplate = jsondata['rows'][1]
    consumertemplate = jsondata['rows'][2]
    jsondata['rows'].pop(2)
    jsondata['rows'].pop(1)

    process_first_row(jsondata['rows'][0], environname)

    if 'topics' in jsoninput:
        add_topics(jsondata, jsoninput['topics'], topictemplate, environname)
    if 'consumergroups' in jsoninput:
        grupts = jsoninput['consumergroups']
        for group in grupts:
            add_consumer_group(jsondata, group, grupts[group], consumertemplate, environname)
    print(json.dumps(jsondata, indent=4, sort_keys=True))


main()
