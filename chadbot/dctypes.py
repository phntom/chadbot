from collections import OrderedDict
from dataclasses import dataclass, field
from typing import OrderedDict as OrderedDictType, Optional, List


@dataclass
class MultiLingualString:
    en: Optional[str] = None
    he: Optional[str] = None


@dataclass
class Answer:
    id: str
    text: MultiLingualString = field(default_factory=MultiLingualString)


@dataclass
class Question:
    id: str
    pre: List[MultiLingualString] = field(default_factory=list)
    q: MultiLingualString = field(default_factory=MultiLingualString)
    a: OrderedDictType[str, Answer] = field(default_factory=OrderedDict)
