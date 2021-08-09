from collections import OrderedDict, defaultdict
from dataclasses import dataclass, field
from typing import OrderedDict as OrderedDictType, DefaultDict, List, Dict

from chadbot.dctypes import Question


@dataclass
class GlobalState:
    questions: OrderedDictType[str, Question] = field(default_factory=OrderedDict)
    specials: DefaultDict[str, List[Question]] = field(default_factory=defaultdict)
    from_actions: Dict[str, Question] = field(default_factory=dict)


g = GlobalState()
g.specials.default_factory = list

