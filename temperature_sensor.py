from faker import Faker

fake = Faker()

def mockup_temperature(left_digits,right_digits,min_value,max_value) -> float:
    """Generates temperature mockup data
    Args: 
        -left_digits: used to craete float range
        -right_digits: used to create float range
    """
    mockup_temperature = fake.pyfloat(
        left_digits=left_digits,
        right_digits=right_digits,
        min_value=min_value,
        max_value=max_value,
    )
    return mockup_temperature