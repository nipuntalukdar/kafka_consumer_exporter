import json
import copy

def add_topics(jsond, topics, template, environment):
    for topic in topics:
        tpl = copy.deepcopy(template)
        tpl['title'] = "Topic:" + topic
        panels = tpl['panels']
        panels[0]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{TOPIC}}}', topic)
        panels[0]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        panels[0]['title'] = panels[0]['title'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][0]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        panels[1]['targets'][1]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{TOPIC}}}', topic)
        panels[1]['targets'][1]['expr'] =\
            panels[0]['targets'][0]['expr'].replace('{{{ENVIRONMENT}}}', environment)
        jsond['rows'].append(tpl)


def add_consumer_group(jsond, group, topics, template, environment):
    for topic in topics:
        tpl = copy.deepcopy(template)
        tpl['title'] = 'Group:{} Topic:{}'.format(group, topic)
        for panel in tpl['panels']:
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

jsondata = json.load(open('template.json', 'r'))
jsoninput = json.load(open('input.json', 'r'))
topictemplate = jsondata['rows'][1]
consumertemplate = jsondata['rows'][2]
jsondata['rows'].pop(2)
jsondata['rows'].pop(1)

if 'topics' in jsoninput:
    add_topics(jsondata, jsoninput['topics'], topictemplate, "myenv")
if 'consumergroups' in jsoninput:
    grupts = jsoninput['consumergroups']
    for group in grupts:
        add_consumer_group(jsondata, group, grupts[group], consumertemplate, 'myenv')
print json.dumps(jsondata['rows'], indent=4, sort_keys=True)
